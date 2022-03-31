package initCyclops

import (
	"M1/Network/API/app"
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/pkg/sftp"
)

const (
	sftpUser      = "tpuser"
	sftpPass      = "tpuser"
	sftpHost      = "10.8.1.1"
	sftpPort      = "22"
	distantFolder = "./use"
	backupFolder  = "./bck"
	redisIP       = sftpHost + ":6379"
)

func InitFile() {
	// Create a url
	rawurl := fmt.Sprintf("sftp://%v:%v@%v", sftpUser, sftpPass, sftpHost)

	// Parse the URL
	parsedUrl, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("Failed to parse SFTP To Go URL: %s", err)
	}

	// Get user name and pass
	user := parsedUrl.User.Username()
	pass, _ := parsedUrl.User.Password()

	// Parse Host and Port
	host := parsedUrl.Host

	// Get hostkey
	//hostKey := getHostKey(host)

	log.Printf("Connecting to %s ...\n", host)

	var auths []ssh.AuthMethod

	// Try to use $SSH_AUTH_SOCK which contains the path of the unix file socket that the sshd agent uses
	// for communication with other processes.
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}

	// Use password authentication if provided
	if pass != "" {
		auths = append(auths, ssh.Password(pass))
	}

	// Initialize client configuration
	config := ssh.ClientConfig{
		User: user,
		Auth: auths,
		// Auth: []ssh.AuthMethod{
		//  ssh.KeyboardInteractive(SshInteractive),
		// },

		// Uncomment to ignore host key check
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		// HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		//  return nil
		// },
		Timeout: 30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", host, sftpPort)

	// Connect to server
	conn, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		log.Fatalf("failed to connect to host [%s]: %v", addr, err)
	}

	defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatalf("unable to start SFTP subsystem: %v", err)
	}
	defer sc.Close()

	// List files in the root directory .
	theFiles, err := listFiles(".")
	if err != nil {
		log.Fatalf("failed to list files in %s: %v", distantFolder, err)
	}

	deleteFiles(*sc, distantFolder)
	app.Drop(redisIP)
	
	for _, file := range theFiles {
		localfile := backupFolder + file.Name
		remoteFile := distantFolder + file.Name
		hash, err := backupFile(*sc, localfile, remoteFile)
		if err != nil {
			log.Fatalf("failed to backup file in %s: %v", distantFolder, err)
		}
		app.Insert(file.Name, hash, redisIP)
	}
}

func SshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	// Hack, check https://stackoverflow.com/questions/47102080/ssh-in-go-unable-to-authenticate-attempted-methods-none-no-supported-method
	answers = make([]string, len(questions))
	// The second parameter is unused
	for n, _ := range questions {
		answers[n] = sftpPass
	}

	return answers, nil
}

type remoteFiles struct {
	Name    string
	Size    string
	ModTime string
}

func listFiles(directory string) (theFiles []remoteFiles, err error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return theFiles, fmt.Errorf("Unable to list local dir in %s : %v", directory, err)
	}

	for _, f := range files {
		var name, modTime, size string

		name = f.Name()
		modTime = f.ModTime().Format("2006-01-02 15:04:05")
		size = fmt.Sprintf("%12d", f.Size())

		if f.IsDir() {
			name = name + "/"
			modTime = ""
			size = "PRE"
		}

		theFiles = append(theFiles, remoteFiles{
			Name:    name,
			Size:    size,
			ModTime: modTime,
		})
	}

	return theFiles, nil
}

func deleteFiles(sc sftp.Client, remoteDir string) {
	files, err := sc.ReadDir(remoteDir)

	if err != nil {
		log.Fatalf("%v", fmt.Errorf("unable to list remote dir for deletion in %s : %v", remoteDir, err))
	}

	for _, f := range files {
		if sc.Remove(f.Name()) != nil {
			log.Fatalf("%v", fmt.Errorf("unable to remove file %s", f.Name()))
		}
	}
}

// Upload file to sftp server
func backupFile(sc sftp.Client, localFile, remoteFile string) (hash string, err error) {
	log.Printf("Uploading [%s] to [%s] ...", localFile, remoteFile)

	srcFile, err := os.Open(localFile)
	if err != nil {
		return "", fmt.Errorf("Unable to open local file: %v", err)
	}
	defer srcFile.Close()

	// Make remote directories recursion
	parent := filepath.Dir(remoteFile)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		sc.Mkdir(path)
	}

	// Note: SFTP Go doesn't support O_RDWR mode
	dstFile, err := sc.OpenFile(remoteFile, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		return "", fmt.Errorf("Unable to open remote file: %v", err)
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("Unable to upload local file: %v", err)
	}
	log.Printf("%d bytes copied", bytes)


	hashObj := sha256.New()
	if _, err := io.Copy(hashObj, srcFile); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%x", hashObj.Sum(nil))

	return string(hashObj.Sum(nil)), nil
}

// Get host key from local known hosts
func getHostKey(host string) ssh.PublicKey {
	// parse OpenSSH known_hosts file
	// ssh or use ssh-keyscan to get initial key
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read known_hosts file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], host) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing %q: %v\n", fields[2], err)
				os.Exit(1)
			}
			break
		}
	}

	if hostKey == nil {
		fmt.Fprintf(os.Stderr, "No hostkey found for %s", host)
		os.Exit(1)
	}

	return hostKey
}