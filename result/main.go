package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
)

type Request struct {
	Message string `json:"message"`
}

/*
type GetInfo struct {
	Pubkey string `json:"identity_pubkey"`
}

type Invoice struct {
	Keysend string `json:"is_keysend"`
	Value   string `json:"value"`
	Settled string `json:"settled"`
}
*/

func main() {
	go listenForPayments()
	fmt.Print("Starting server on port 8080\n")
	server()
}

func writeToBoard(content string) error {
	f, err := os.OpenFile("database.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s\n", content)); err != nil {
		return err
	}
	return nil
}

func server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			content, err := ioutil.ReadFile("database.txt")
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(w, string(content))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != http.ErrServerClosed {
		fmt.Print("Server started on port 8080")
		panic(err)
	}
}

func listenForPayments() error {
	grpcConn, err := grpcSetup()
	if err != nil {
		fmt.Printf(err.Error())
	}
	defer grpcConn.Close()
	lncli := lnrpc.NewLightningClient(grpcConn)

	ctx, _ := context.WithCancel(context.Background())
	getInfoReq := lnrpc.GetInfoRequest{}
	infoRes, err := lncli.GetInfo(ctx, &getInfoReq)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Listening on node's pubkey: %v\n---\n", infoRes.IdentityPubkey)

	invoiceStream, err := lncli.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{})
	if err != nil {
		fmt.Printf("Error listening for Invoice Stream: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		invoiceEvent, err := invoiceStream.Recv()
		if errors.Is(err, io.EOF) {
			log.Println("EOF when processing invoice streams")
			break
		}

		time.Sleep(1)
		if invoiceEvent.Settled {
			fmt.Printf("Invoice was settled! Settled at %+v\n", invoiceEvent.SettleDate)
			for _, htlc := range invoiceEvent.Htlcs {
				customMessage := string(htlc.CustomRecords[420420])
				if customMessage != "" {
					fmt.Printf("Found a message in TLV: %+v\n\n", customMessage)
					err = writeToBoard(customMessage)
					if err != nil {
						fmt.Printf("There was a problem writing post to board!: %+v", err)
					}
				} else {
					fmt.Printf("No custom message was found\n\n")
				}
			}
		}

	}
	return nil
}

func grpcSetup() (*grpc.ClientConn, error) {

	//host := "bbbbb.t.voltageapp.io:10009"
	host := "127.0.0.1:10002"

	certFileLocation := fmt.Sprintf("../tls.cert")
	tlsContent, err := ioutil.ReadFile(certFileLocation)
	if err != nil {
		panic(err)
	}

	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(tlsContent) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}
	tlsCreds := credentials.NewClientTLSFromCert(cp, "")

	macaroonFileLocation := fmt.Sprintf("../bbbbb.macaroon")
	content, err := ioutil.ReadFile(macaroonFileLocation)
	if err != nil {
		panic(err)
	}

	mac := &macaroon.Macaroon{}
	if err := mac.UnmarshalBinary(content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal macaroon: %v", err)
	}

	macCred, err := macaroons.NewMacaroonCredential(mac)
	if err != nil {
		return nil, fmt.Errorf("cannot create macaroon credentials: %v", err)
	}

	opts := []grpc.DialOption{
		grpc.WithReturnConnectionError(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(tlsCreds),
		grpc.WithPerRPCCredentials(macCred),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	conn, err := grpc.DialContext(ctx, host, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot dial to lnd: %v", err)
	}

	return conn, nil
}
