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
	"fmt"
	"github.com/gen2brain/dlgs"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
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

func pemBlockOf(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
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
			Organization: []string{"The Postwoman Project"},
		},
		NotBefore: time.Now(),
		// Make certificate expire after 10 years.
		NotAfter:  time.Now().Add(time.Hour * 24 * 3650),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"));

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKeyOf(private), private);
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err);
	}

	keypair := new([2]bytes.Buffer);

	certificatePairItem := &bytes.Buffer{};
	_ = pem.Encode(certificatePairItem, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes});
	keypair[0] = *certificatePairItem;

	privatePairItem := &bytes.Buffer{};
	privBytes, _ := x509.MarshalPKCS8PrivateKey(private);
	_ = pem.Encode(privatePairItem, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes });
	keypair[1] = *privatePairItem;

	return keypair;
}

func EnsurePrivateKeyInstalled () error {
	_, err := os.Stat(GetDataPath() + "/cert.pem");
	_, err = os.Stat(GetDataPath() + "/key.pem");

	// If the error is that the file does not exist, create the file
	// and then return no error (unless one was thrown in the process of creating the key.)
	if(os.IsNotExist(err)){
		encodedPEM := CreateKeyPair();
		err = ioutil.WriteFile(GetDataPath() + "/cert.pem", encodedPEM[0].Bytes(), 0600);
		err = ioutil.WriteFile(GetDataPath() + "/key.pem", encodedPEM[1].Bytes(), 0600);

		if runtime.GOOS == "darwin" {
			_ = exec.Command("open", GetDataPath()).Run();
			_, _ = dlgs.Warning("Postwoman Proxy", "Postwoman needs you to install a root certificate authority (cert.pem).\nPlease double-click the certificate file to open it in Keychain Access and follow the installation and trust process.\n\nFor more information about this process and why it's required, please click the Postwoman icon in the status tray and select 'Help'.\n\nClick OK when you have installed the certificate and marked it as trusted.");
		}

		return err;
	}
	// Otherwise return any errors that may have occurred.
	// (This is nil if no errors occurred.)
	return err;
}

func LoadKeyPair() error {
	encodedPem, _ := ioutil.ReadFile(GetDataPath() + "/cert.pem");
	block, _ := pem.Decode(encodedPem);

	if(block.Type != "CERTIFICATE"){
		return nil;
	}

	//derBytes := block.Bytes;
	//_, _  := x509.ParsePKCS1PrivateKey(derBytes);
	return nil;
}

func GetDataPath() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]));
	return dir + "/data";
}