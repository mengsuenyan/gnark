// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"time"

	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gurvy"
	"github.com/google/uuid"

	"github.com/consensys/gnark/gnarkd/circuits/bn256/cubic"
	"github.com/consensys/gnark/gnarkd/pb"
	"google.golang.org/grpc"
)

const (
	address = "localhost:9002"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGroth16Client(conn)

	// Contact the server and print out its response.
	ctx := context.Background()

	var w cubic.Circuit
	w.X.Assign(3)
	w.Y.Assign(35)
	var buf bytes.Buffer
	err = witness.WriteFull(&buf, &w, gurvy.BN256)
	if err != nil {
		log.Fatalf("serializing witness: %v", err)
	}

	_, err = c.Prove(ctx, &pb.ProveRequest{
		CircuitID: "bn256/cubic",
		Witness:   buf.Bytes(),
	})
	if err != nil {
		log.Fatalf("could not prove: %v", err)
	}
	log.Println("proof ok")

	r, err := c.CreateProveJob(ctx, &pb.CreateProveJobRequest{CircuitID: "bn256/cubic"})
	if err != nil {
		log.Fatalf("could not create job: %v", err)
	}
	log.Println("job id:", r.JobID)

	stream, err := c.SubscribeToProveJob(ctx, &pb.SubscribeToProveJobRequest{JobID: r.JobID})
	if err != nil {
		log.Fatalf("could not subscribe to job: %v", err)
	}

	done := make(chan struct{})

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- struct{}{}
				return
			}
			if err != nil {
				log.Fatalf("cannot receive %v", err)
			}
			log.Printf("Resp received: %s", resp.Status.String())
			if resp.Status == pb.ProveJobResult_ERRORED {
				log.Fatalf("with error %s", *resp.Err)
			}
		}
	}()
	<-time.After(4 * time.Second)
	go func() {
		// send witness
		conn, err := net.Dial("tcp", "127.0.0.1:9001")
		// set conn.Deadlines
		defer conn.Close()

		if err != nil {
			log.Fatalf("cannot connect to witness socket %v", err)
			return
		}

		jobID, err := uuid.Parse(r.JobID)
		if err != nil {
			panic(err)
		}
		bjobID, err := jobID.MarshalBinary()
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(bjobID)
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(buf.Bytes())
		if err != nil {
			panic(err)
		}
	}()

	<-done //we will wait until all response is received
}
