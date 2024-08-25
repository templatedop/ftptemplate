package cron

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func connectSftp(rawurl string) (*sftp.Client, *ssh.Client) {
	parsedUrl, err := url.Parse(rawurl)
	if err != nil {
		fmt.Println("Failed to parse SFTP To Go URL:", err)
		//log.Error( "Failed to parse SFTP To Go URL:", err)
		os.Exit(1)
	}

	// Get user name and pass
	user := parsedUrl.User.Username()
	pass, _ := parsedUrl.User.Password()

	// Parse Host and Port
	host := parsedUrl.Host
	// Default SFTP port
	port := 22

	//hostKey := getHostKey(host)

	//log.Info("Connecting to %s ... at %v", host, time.Now())

	fmt.Fprintf(os.Stdout, "Connecting to %s ... at %v \n", host, time.Now())

	var auths []ssh.AuthMethod

	var aconn net.Conn
	if runtime.GOOS == "windows" {
		if aconn, err = net.Dial("unix", `\\.\pipe\openssh-ssh-agent`); err == nil {
			auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
		}
	} else {
		if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
		} else {
			// Handle error
			panic(err)
		}
	}

	if pass != "" {
		auths = append(auths, ssh.Password(pass))
	}

	// Initialize client configuration
	config := ssh.ClientConfig{
		User: user,
		Auth: auths,

		//HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// Connect to server
	conn, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connecto to [%s]: %v\n", addr, err)
		os.Exit(1)
	}

	//defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start SFTP subsystem: %v\n", err)
		os.Exit(1)
	}

	//listFiles(sc, "/IT2/TO_CSI")
	//defer sc.Close()
	return sc, conn

}

func listSftpFiles(sc *sftp.Client, remoteDir string) ([]fs.FileInfo, error) {
	fmt.Fprintf(os.Stdout, "Listing [%s] ...\n\n", remoteDir)

	files, err := sc.ReadDir(remoteDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to list remote dir: %v\n", err)
		return nil, err
	}
	var newfiles []fs.FileInfo
	for _, f := range files {

		if !f.IsDir() {
			newfiles = append(newfiles, f)
		}

	}

	return newfiles, nil
}

func listFiles(sc *sftp.Client, remoteDir string) (files []fs.FileInfo, err error) {
	fmt.Fprintf(os.Stdout, "Listing [%s] ...\n\n", remoteDir)

	files, err = sc.ReadDir(remoteDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to list remote dir: %v\n", err)
		return
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
		// Output each file name and size in bytes
		fmt.Fprintf(os.Stdout, "%19s %12s %s\n", modTime, size, name)
	}

	return
}

func uploadFile(sc *sftp.Client, localFile, remoteFile string) (err error) {
	fmt.Fprintf(os.Stdout, "Uploading [%s] to [%s] ...\n", localFile, remoteFile)

	srcFile, err := os.Open(localFile)
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		return fmt.Errorf("unable to open local file: %w", err)
		// return
	}
	defer srcFile.Close()

	// Make remote directories recursion
	//parent := filepath.Dir(remoteFile)
	//  path := string(filepath.Separator)
	// dirs := strings.Split(parent, path)
	// for _, dir := range dirs {
	//     path = filepath.Join(path, dir)
	//     // if err := sc.Mkdir(path); err != nil && !os.IsExist(err) {
	//     //     return fmt.Errorf("unable to create remote directory: %w", err)
	//     // }
	//     sc.Mkdir(path)
	// }

	// Note: SFTP To Go doesn't support O_RDWR mode
	dstFile, err := sc.OpenFile(remoteFile, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		return fmt.Errorf("unable to open remote file: %w", err)
		//return
	}
	defer dstFile.Close()
	startTime := time.Now()
	buf := make([]byte, 32*1024)
	bytes, err := io.CopyBuffer(dstFile, srcFile, buf)
	//bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("unable to upload local file: %w", err)
		// fmt.Fprintf(os.Stderr, "Unable to upload local file: %v\n", err)
		//os.Exit(1)
	}
	elapsedTime := time.Since(startTime)
	// fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)
	fmt.Printf("%d bytes copied in %s\n", bytes, elapsedTime)

	return nil
}

func downloadFile(sc *sftp.Client, remoteFile, localFile string) (err error) {

	fmt.Fprintf(os.Stdout, "Downloading [%s] to [%s] ...\n", remoteFile, localFile)
	// Note: SFTP To Go doesn't support O_RDWR mode
	srcFile, err := sc.OpenFile(remoteFile, (os.O_RDONLY))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open remote file: %v\n", err)
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open local file: %v\n", err)
		return
	}
	defer dstFile.Close()
	buf := make([]byte, 32*1024)
	bytes, err := io.CopyBuffer(dstFile, srcFile, buf)
	//bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to download remote file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%d bytes copied\n", bytes)

	return
}

func moveFile(sc *sftp.Client, sourcePath, destinationPath string) error {
	err := checkpath(sc, sourcePath)
	if err != nil {
		return fmt.Errorf("unable to check source file [%s]: %w", sourcePath, err)
	}

	// if _, err := sc.Stat(sourcePath); err != nil {
	// 	if os.IsNotExist(err) {
	// 		return fmt.Errorf("source file [%s] does not exist: %w", sourcePath, err)
	// 	}
	// 	return fmt.Errorf("unable to stat source file [%s]: %w", sourcePath, err)
	// }

	err = checkfolder(sc, destinationPath)
	if err != nil {
		return fmt.Errorf("unable to check folder [%s]: %w", destinationPath, err)
	}

	err = sc.Rename(sourcePath, destinationPath)
	if err != nil {
		return fmt.Errorf("unable to move file from [%s] to [%s]: %w", sourcePath, destinationPath, err)
	}
	fmt.Printf("File moved from [%s] to [%s]\n", sourcePath, destinationPath)
	return nil
}

func checkfolder(sc *sftp.Client, path string) (err error) {
	folder := filepath.Dir(path)
	_, err = sc.Stat(strings.ReplaceAll(folder, "\\", "/"))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("folder [%s] does not exist: %w", folder, err)
		}
		return fmt.Errorf("unable to stat folder [%s]: %w", folder, err)
	}
	return nil
}

func checkpath(sc *sftp.Client, sourcePath string) error {
	if _, err := sc.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source file [%s] does not exist: %w", sourcePath, err)
		}
		return fmt.Errorf("unable to stat source file [%s]: %w", sourcePath, err)
	}
	return nil
}

func listLocalFiles(directory string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var files []fs.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry)
		}
	}
	return files, nil
}

func moveLocalFile(sourcePath, destinationPath, filename string) error {
	// Ensure the destination directory exists
	// err := os.MkdirAll(filepath.Dir(destinationPath), os.ModePerm)
	// if err != nil {
	//     return fmt.Errorf("unable to create destination directory: %w", err)
	// }

	// Move the file
	err := os.Rename(sourcePath+"/"+filename, destinationPath+"/"+filename)
	if err != nil {
		return fmt.Errorf("unable to move file from [%s] to [%s]: %w", sourcePath, destinationPath, err)
	}

	fmt.Println("File successfully moved")
	return nil
}
