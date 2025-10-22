package libproxy

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/gen2brain/dlgs"
)

func publicKeyOf(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func CreateKeyPair() *[2]bytes.Buffer {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"The Hoppscotch Project"},
		},
		NotBefore: time.Now(),
		// Make certificate expire after 10 years.
		NotAfter: time.Now().Add(time.Hour * 24 * 3650),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKeyOf(private), private)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	keypair := new([2]bytes.Buffer)

	certificatePairItem := &bytes.Buffer{}
	_ = pem.Encode(certificatePairItem, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keypair[0] = *certificatePairItem

	privatePairItem := &bytes.Buffer{}
	privBytes, _ := x509.MarshalPKCS8PrivateKey(private)
	_ = pem.Encode(privatePairItem, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	keypair[1] = *privatePairItem

	return keypair
}

func EnsurePrivateKeyInstalled() error {
	_, err := os.Stat(GetOrCreateDataPath() + "/cert.pem")
	if !os.IsNotExist(err) {
		_, err = os.Stat(GetOrCreateDataPath() + "/key.pem")
	}
	// If the error is that the file does not exist, create the file
	// and then return no error (unless one was thrown in the process of creating the key.)
	if os.IsNotExist(err) {
		encodedPEM := CreateKeyPair()
		err = os.WriteFile(GetOrCreateDataPath()+"/cert.pem", encodedPEM[0].Bytes(), 0600)

		// There's no point writing the key if we failed to write the certificate, so only do that
		// if there is no error.
		if err == nil {
			err = os.WriteFile(GetOrCreateDataPath()+"/key.pem", encodedPEM[1].Bytes(), 0600)
		}

		if runtime.GOOS == "windows" {
			// Windows doesn't recognize .pem as certificates, but we can simply write the PEM data
			// into a .cer file and it works just fine!
			err = os.WriteFile(GetOrCreateDataPath()+"/cert.cer", encodedPEM[0].Bytes(), 0600)
		}

		// Assuming that there was no error writing either the certificate, or the key, we can continue
		// to prompt the user to install the certificate authority.
		if err != nil {
			if runtime.GOOS == "darwin" {
				_ = exec.Command("open", GetOrCreateDataPath()).Run()
				_, err = dlgs.Warning("Proxyscotch", "Proxyscotch needs you to install a root certificate authority (cert.pem).\nPlease double-click the certificate file to open it in Keychain Access and follow the installation and trust process.\n\nFor more information about this process and why it's required, please click the Hoppscotch icon in the status tray and select 'Help'.\n\nClick OK when you have installed the certificate and marked it as trusted.")
			}

			if runtime.GOOS == "windows" {
				_ = exec.Command("explorer.exe", GetOrCreateDataPath()+string(os.PathSeparator)+"cert.cer").Run()
				_, err = dlgs.Warning("Proxyscotch", "Proxyscotch needs you to install a root certificate authority (cert.cer).\nPlease install the certificate (opened) into the 'Trusted Root Certification Authorities' store for the Local Machine.\n\nFor more information about this process and why it's required, please click the Hoppscotch icon in the system tray and select 'Help'.\n\nClick OK when you have installed the certificate and marked it as trusted.")
			}

			if runtime.GOOS == "linux" {
				_ = exec.Command("xdg-open", GetOrCreateDataPath()).Run()
				_, err = dlgs.Warning("Proxyscotch", "Proxyscotch needs you to install a root certificate authority (cert.pem).\n[INSTRUCTIONS PENDING]\n\nFor more information about this process and why it's required, please click the Hoppscotch icon in the status tray and select 'Help'.\n\nClick OK when you have installed the certificate and marked it as trusted.")
			}
		}

		return err
	}
	// Otherwise return any errors that may have occurred.
	// (This is nil if no errors occurred.)
	return err
}

func GetOrCreateDataPath() string {
	// Backward compatibility: use the legacy data dir if it contains
	// a cert.pem file
	binDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dataDir := filepath.Join(binDir, "data")
	if _, err := os.Stat(filepath.Join(dataDir, "cert.pem")); err == nil {
		return dataDir
	}

	dataDir = filepath.Join(xdg.DataHome, "proxyscotch")

	// If the data directory stat fails because the direcotry does not exist,
	// create the data directory.
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		_ = os.Mkdir(dataDir, 0700)
	}

	return dataDir
}
