package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	//"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	//"strings"
	"time"

	"github.com/noot/ring-go/ring"

	//"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	//"github.com/ethereum/go-ethereum/accounts"
	//"github.com/ethereum/go-ethereum/accounts/keystore"
)

// generate a new public-private keypair and save in ./keystore directory
func gen() {
	priv, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	pub := priv.Public().(*ecdsa.PublicKey)

	fp, err := filepath.Abs("./keystore")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		os.Mkdir("./keystore", os.ModePerm)
	}

	fp, err = filepath.Abs(fmt.Sprintf("./keystore/%d.priv", time.Now().Unix()))
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(fp, []byte(fmt.Sprintf("%x", priv.D.Bytes())), 0644)
	if err != nil {
		log.Fatal(err)
	}

	name := time.Now().Unix()
	fp, err = filepath.Abs(fmt.Sprintf("./keystore/%d.pub", name))
	if err != nil {
		log.Fatal(err)
	}
	//err = ioutil.WriteFile(fp, []byte(fmt.Sprintf("%x", elliptic.Marshal(crypto.S256(), pub.X, pub.Y))), 0644)
	err = ioutil.WriteFile(fp, elliptic.Marshal(crypto.S256(), pub.X, pub.Y), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("output saved to ./keystore/%d\n", name)
	os.Exit(0)
}

func sign() {
	// read public keys and put them in a ring
	fp, err := filepath.Abs(os.Args[2])
	if err != nil {
		log.Fatal("could not read key from ", os.Args[2], "\n", err)
	}
	files, err := ioutil.ReadDir(fp)
	if err != nil {
		log.Fatal(err)
	}

	if len(files) == 0 {
		log.Fatalf("No public keys from in %s", os.Args[2])
	}

	pubkeys := make([]*ecdsa.PublicKey, len(files))

	for i, file := range files {
		fp, err = filepath.Abs(fmt.Sprintf("%s/%s", os.Args[2], file.Name()))
		key, err := ioutil.ReadFile(fp)
		if err != nil {
			log.Fatal("could not read key from ", fp, "\n", err)
		}

		keyStr := string(key)

		fmt.Printf("%s:%x\n", file.Name(), keyStr)

		pub, err := crypto.UnmarshalPubkey(key)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s.X:%x\n", file.Name(), pub.X)
		fmt.Printf("%s.Y:%x\n", file.Name(), pub.Y)

		pubkeys[i] = pub
	}

	// handle secret key and generate ring of pubkeys
	fp, err = filepath.Abs(os.Args[3])
	privBytes, err := ioutil.ReadFile(fp)
	if err != nil {
		log.Fatal("could not read key from ", fp, "\n", err)
	}

	priv := new(ecdsa.PrivateKey)
	priv.Curve = crypto.S256()
	priv.D = big.NewInt(0).SetBytes(privBytes[0:32])

	priv.PublicKey.Curve = priv.Curve
	priv.PublicKey.X, priv.PublicKey.Y = priv.Curve.ScalarBaseMult(priv.D.Bytes())

	fmt.Printf("secret.pub:%x%x\n", priv.X, priv.Y)

	sb, err := rand.Int(rand.Reader, new(big.Int).SetInt64(int64(len(pubkeys))))
	if err != nil {
		log.Fatal(err)
	}
	s := int(sb.Int64())

	r, err := ring.GenKeyRing(pubkeys, priv, s)
	if err != nil {
		log.Fatal(err)
	}

	// read message and hash
	fp, err = filepath.Abs(os.Args[4])
	msgBytes, err := ioutil.ReadFile(fp)
	if err != nil {
		log.Fatal("could not read key from ", fp, "\n", err)
	}

	msgHash := sha3.Sum256(msgBytes)

	// all good, let's sign
	sig, err := ring.Sign(msgHash, r, priv, s)
	if err != nil {
		log.Fatal(err)
	}

	// save signature
	fmt.Println("Signature successfully generated!")

	fp, err = filepath.Abs("./signatures")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		os.Mkdir("./signatures", os.ModePerm)
	}

	name := time.Now().Unix()
	fp, err = filepath.Abs(fmt.Sprintf("./signatures/%d.sig", name))
	if err != nil {
		log.Fatal(err)
	}

	serializedSig, err := sig.Serialize()
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(fp, []byte(fmt.Sprintf("%x", serializedSig)), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("output saved to ./signatures/%d.sig\n", name)
	os.Exit(0)
}

func main() {
	fmt.Println("welcome to ring-go...")

	// cli options
	genPtr := flag.Bool("gen", false, "generate a new public-private keypair")
	signPtr := flag.Bool("sign", false, "sign a message with a ring signature")
	//messagePtr := flag.String("m", "", "path to message file")
	verifyPtr := flag.Bool("verify", false, "verify a ring signature")

	// if no flags passed, display help
	if len(os.Args) < 2 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.Parse()
	if *genPtr {
		gen()
	}

	if *signPtr {
		if len(os.Args) < 2 {
			fmt.Println("need to supply path to public key directory: ring-go --sign /path/to/pubkey/dir /path/to/privkey.priv message.txt")
			os.Exit(0)
		}

		if len(os.Args) < 3 {
			fmt.Println("need to supply path to private key file: ring-go --sign /path/to/pubkey/dir /path/to/privkey.priv message.txt")
			os.Exit(0)
		}

		if len(os.Args) < 4 {
			fmt.Println("need to supply path to message file: ring-go --sign /path/to/pubkey/dir /path/to/privkey.priv message.txt")
			os.Exit(0)
		}

		sign()
	}

	if *verifyPtr {
		os.Exit(0)
	}

	/* generate new private public keypair */
	// privkey, err := crypto.HexToECDSA("358be44145ad16a1add8622786bef07e0b00391e072855a5667eb3c78b9d3803")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// /* sign message */
	// file, err := ioutil.ReadFile("./message.txt")
	// if err != nil {
	// 	log.Fatal("could not read message from message.txt", err)
	// }
	// msgHash := sha3.Sum256(file)

	// /* secret index */
	// s := 7

	//  generate keyring 
	// keyring := ring.GenNewKeyRing(12, privkey, s)

	// /* sign */
	// sig, err := ring.Sign(msgHash, keyring, privkey, s)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(sig.S)

	// byteSig := sig.SerializeSignature()

	// fmt.Println("signature: ")
	// fmt.Println(fmt.Sprintf("0x%x", byteSig))

	// /* verify signature */
	// ver := ring.Verify(sig)
	// fmt.Println("verified? ", ver)
}
