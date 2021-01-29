package server

import (
	"bytes"
	context "context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/examples/cubic"
	"github.com/consensys/gnark/gnarkd/pb"
	"github.com/consensys/gurvy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var (
	grpcListener    *bufconn.Listener
	witnessListener *bufconn.Listener
	cancelServer    context.CancelFunc
	gnarkdServer    *Server
)

// -------------------------------------------------------------------------------------------------
// logger
var (
	logger *zap.Logger
	log    *zap.SugaredLogger
)

func init() {
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		fmt.Println("unable to create logger")
		os.Exit(1)
	}
	log = logger.Sugar()
}

func setupServer() {
	grpcListener = bufconn.Listen(bufSize)
	witnessListener = bufconn.Listen(bufSize)
	s := grpc.NewServer()

	var serverCtx context.Context
	var err error
	serverCtx, cancelServer = context.WithCancel(context.Background())
	gnarkdServer, err = NewServer(serverCtx, log, "../circuits")
	if err != nil {
		log.Fatalw("couldn't init gnarkd", "err", err)
	}

	// start witness listener
	go gnarkdServer.StartWitnessListener(witnessListener)
	pb.RegisterGroth16Server(s, gnarkdServer)

	go func() {
		if err := s.Serve(grpcListener); err != nil {
			log.Fatalw("Server exited with error", "err", err)
		}
	}()
}

func shutdownServer() {
	grpcListener.Close()
	witnessListener.Close()
	cancelServer()
	cancelServer = nil
	grpcListener = nil
	witnessListener = nil
	gnarkdServer = nil
}

func TestMain(m *testing.M) {
	setupServer()
	code := m.Run()
	shutdownServer()
	os.Exit(code)
}

func TestProveSync(t *testing.T) {
	assert := require.New(t)

	// create grpc client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(
		func(c context.Context, s string) (net.Conn, error) {
			return grpcListener.Dial()
		}), grpc.WithInsecure())

	assert.NoError(err)
	defer conn.Close()

	c := pb.NewGroth16Client(conn)

	// 1. serialize a valid witness
	var (
		w        cubic.Circuit
		bWitness bytes.Buffer
	)
	w.X.Assign(3)
	w.Y.Assign(35)

	err = witness.WriteFull(&bWitness, &w, gurvy.BN256)
	assert.NoError(err)

	// 2. call prove
	proveResult, err := c.Prove(ctx, &pb.ProveRequest{
		CircuitID: "bn256/cubic",
		Witness:   bWitness.Bytes(),
	})
	assert.NoError(err, "grpc sync prove failed")

	// 3. ensure returned proof is valid.
	proof := groth16.NewProof(gurvy.BN256)
	_, err = proof.ReadFrom(bytes.NewReader(proveResult.Proof))
	assert.NoError(err, "deserializing grpc proof response failed")

	err = groth16.Verify(proof, gnarkdServer.circuits["bn256/cubic"].vk, &w)
	assert.NoError(err, "couldn't verify proof returned from grpc server")

	// 4. create invalid proof
	var wBad cubic.Circuit
	wBad.X.Assign(4)
	wBad.Y.Assign(42)
	bWitness.Reset()
	err = witness.WriteFull(&bWitness, &wBad, gurvy.BN256)
	assert.NoError(err)
	proveResult, err = c.Prove(ctx, &pb.ProveRequest{
		CircuitID: "bn256/cubic",
		Witness:   bWitness.Bytes(),
	})
	assert.Error(err, "grpc sync false prove failed")
}

func TestProveAsync(t *testing.T) {
	assert := require.New(t)

	// create grpc client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(
		func(c context.Context, s string) (net.Conn, error) {
			return grpcListener.Dial()
		}), grpc.WithInsecure())

	assert.NoError(err)
	defer conn.Close()

	client := pb.NewGroth16Client(conn)

	// 1. serialize a valid witness
	var (
		w        cubic.Circuit
		bWitness bytes.Buffer
	)
	w.X.Assign(3)
	w.Y.Assign(35)

	err = witness.WriteFull(&bWitness, &w, gurvy.BN256)
	assert.NoError(err)

	// 2. call prove
	r, err := client.CreateProveJob(ctx, &pb.CreateProveJobRequest{
		CircuitID: "bn256/cubic",
	})
	assert.NoError(err, "grpc sync create prove failed")

	// 3. subscribe to status changes
	stream, err := client.SubscribeToProveJob(ctx, &pb.SubscribeToProveJobRequest{JobID: r.JobID})
	assert.NoError(err, "couldn't subscribe to job")

	done := make(chan struct{})
	var lastStatus pb.ProveJobResult_Status
	var rproof []byte
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- struct{}{}
				return
			}
			assert.NoError(err)
			assert.NotEqual(pb.ProveJobResult_ERRORED, resp.Status, "we don't expect the job to produce error")
			lastStatus = resp.Status
			if lastStatus == pb.ProveJobResult_COMPLETED {
				rproof = resp.Proof
			}
		}
	}()

	// 4. send wtness on the wire
	wc, err := witnessListener.Dial()
	assert.NoError(err, "dialing witness socket")
	defer wc.Close()
	jobID, err := uuid.Parse(r.JobID)
	assert.NoError(err)
	bjobID, err := jobID.MarshalBinary()
	assert.NoError(err)
	_, err = wc.Write(bjobID)
	assert.NoError(err)
	_, err = wc.Write(bWitness.Bytes())
	assert.NoError(err)

	<-done
	assert.Equal(lastStatus, pb.ProveJobResult_COMPLETED)

	// 3. ensure returned proof is valid.
	proof := groth16.NewProof(gurvy.BN256)
	_, err = proof.ReadFrom(bytes.NewReader(rproof))
	assert.NoError(err, "deserializing grpc proof response failed")

	err = groth16.Verify(proof, gnarkdServer.circuits["bn256/cubic"].vk, &w)
	assert.NoError(err, "couldn't verify proof returned from grpc server")

}

func TestVerifySync(t *testing.T) {
	assert := require.New(t)

	// create grpc client connection
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithContextDialer(
		func(c context.Context, s string) (net.Conn, error) {
			return grpcListener.Dial()
		}), grpc.WithInsecure())

	assert.NoError(err)
	defer conn.Close()

	client := pb.NewGroth16Client(conn)

	// 1. serialize a valid witness
	var (
		w        cubic.Circuit
		bWitness bytes.Buffer
		bProof   bytes.Buffer
	)
	w.X.Assign(3)
	w.Y.Assign(35)
	proof, err := groth16.Prove(gnarkdServer.circuits["bn256/cubic"].r1cs, gnarkdServer.circuits["bn256/cubic"].pk, &w)
	assert.NoError(err)
	_, err = proof.WriteRawTo(&bProof)
	assert.NoError(err)

	err = witness.WritePublic(&bWitness, &w, gurvy.BN256)
	assert.NoError(err)

	// 2. call verify
	vResult, err := client.Verify(ctx, &pb.VerifyRequest{
		CircuitID:     "bn256/cubic",
		PublicWitness: bWitness.Bytes(),
		Proof:         bProof.Bytes(),
	})
	assert.NoError(err, "grpc sync verify failed")
	assert.True(vResult.Ok)
}
