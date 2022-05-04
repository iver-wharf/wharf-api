package main

import (
	"io"
	"math"
	"net"
	"time"

	v5 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v5"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/typ.v4/chans"
	"gorm.io/gorm"
)

type grpcWharfServer struct {
	v5.UnimplementedBuildsServer
	db *gorm.DB
}

func serveGRPC(listener net.Listener, db *gorm.DB) {
	grpcServer := grpc.NewServer()
	grpcWharf := &grpcWharfServer{db: db}
	v5.RegisterBuildsServer(grpcServer, grpcWharf)
	grpcServer.Serve(listener)
}

func (s *grpcWharfServer) CreateLogStream(stream v5.Builds_CreateLogStreamServer) error {
	logReqChan := make(chan *v5.CreateLogStreamRequest, 10)
	var streamErr error
	go func() {
		streamErr = recvLogStreamIntoChan(stream, logReqChan)
	}()

	var logsInserted uint64
	for {
		const bufferSize = 100
		lines := recvQueuedAtLeastOne(logReqChan, bufferSize)
		if len(lines) == 0 {
			break
		}
		dbLogs := make([]database.Log, 0, len(lines))
		for _, line := range lines {
			if err := line.Timestamp.CheckValid(); err != nil {
				log.Warn().WithError(err).
					Message("Received invalid log timestamp, skipping.")
				continue
			}
			if line.BuildID == 0 {
				log.Warn().Message("Received log with build ID: 0, skipping.")
				continue
			}
			if line.BuildID > math.MaxUint {
				log.Warn().WithUint64("buildId", line.BuildID).
					Message("Received too big log build ID, skipping.")
				return status.Errorf(codes.InvalidArgument,
					"received build ID is too big: %d (build ID) > %d (max)",
					line.BuildID, uint(math.MaxUint))
			}
			dbLogs = append(dbLogs, database.Log{
				BuildID:   uint(line.BuildID),
				Message:   line.Message,
				Timestamp: line.Timestamp.AsTime(),
			})
		}
		log.Debug().WithInt("lines", len(lines)).Message("Received log lines")
		if len(dbLogs) == 0 {
			continue
		}
		createdLogs, err := createLogBatch(s.db.WithContext(stream.Context()), dbLogs)
		if err != nil {
			return status.Errorf(codes.Internal, "insert logs: %v", err)
		}
		log.Debug().WithInt("created", len(createdLogs)).Message("Inserted logs into database")
		for _, dbLog := range createdLogs {
			build(dbLog.BuildID).Submit(response.Log{
				LogID:     dbLog.LogID,
				BuildID:   dbLog.BuildID,
				Message:   dbLog.Message,
				Timestamp: dbLog.Timestamp,
			})
		}
		logsInserted += uint64(len(createdLogs))
		time.Sleep(time.Second)
	}

	if streamErr != nil {
		return streamErr
	}
	err := stream.SendAndClose(&v5.CreateLogStreamResponse{
		LinesInserted: logsInserted,
	})
	if err != nil {
		return err
	}
	return nil
}

func recvLogStreamIntoChan(stream v5.Builds_CreateLogStreamServer, ch chan<- *v5.CreateLogStreamRequest) error {
	defer close(ch)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		} else if msg == nil {
			return status.Error(codes.InvalidArgument, "received nil message")
		}
		ch <- msg
	}
}

func recvQueuedAtLeastOne[C chans.Receiver[E], E any](ch C, bufferSize int) []E {
	buf := make([]E, 100)
	firstMsg, ok := <-ch
	if !ok {
		return nil
	}
	buf[0] = firstMsg
	count := chans.RecvQueuedFull(ch, buf[1:])
	return buf[:count+1]
}
