package sftp

import (
	"fmt"
	"path/filepath"

	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Destination struct {
	PrivateKey             string `credential:"true" mapstructure:"private_key" description:"SSH Private Key"`
	User                   string `mapstructure:"user" description:"Username"`
	Host                   string `mapstructure:"host" description:"Hostname"`
	Port                   string `mapstructure:"port" description:"Port"`
	CertificateDestination string `mapstructure:"certificate_destination" description:"Full path, including file name, for certificate"`
	PrivateKeyDestination  string `mapstructure:"private_key_destination" description:"Full path, including file name, for private key"`
}

func (d Destination) Description() string {
	return "Uploads certificate via SFTP to a remote directory"
}

func (d Destination) Upload(request models.CertRequest, cert *certificate.Resource) error {
	// Parse private key
	signer, err := ssh.ParsePrivateKey([]byte(d.PrivateKey))
	if err != nil {
		return err
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User: d.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: Avoid using InsecureIgnoreHostKey in production
	}

	// Connect to SSH
	address := fmt.Sprintf("%s:%s", d.Host, d.Port)
	sshClient, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	dir, _ := filepath.Split(d.CertificateDestination)
	err = sftpClient.MkdirAll(dir)
	if err != nil {
		return err
	}

	remoteCertFile, err := sftpClient.Create(d.CertificateDestination)
	if err != nil {
		return err
	}
	defer remoteCertFile.Close()

	_, err = remoteCertFile.Write(cert.Certificate)
	if err != nil {
		return err
	}

	dir, _ = filepath.Split(d.PrivateKeyDestination)
	err = sftpClient.MkdirAll(dir)
	if err != nil {
		return err
	}

	remotePrivKey, err := sftpClient.Create(d.PrivateKeyDestination)
	if err != nil {
		return err
	}
	defer remotePrivKey.Close()

	_, err = remotePrivKey.Write(cert.PrivateKey)
	if err != nil {
		return err
	}

	return nil
}
