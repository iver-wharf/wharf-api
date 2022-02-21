package main

import (
	"io"
	"math"

	v5 "github.com/iver-wharf/wharf-api/v5/api/wharfapi/v5"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/database"
	"github.com/iver-wharf/wharf-api/v5/pkg/model/response"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type grpcWharfServer struct {
	v5.UnimplementedBuildsServer
	db *gorm.DB
}

func (s *grpcWharfServer) CreateLogStream(stream v5.Builds_CreateLogStreamServer) error {
	var logsInserted uint64
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			stream.SendAndClose(&v5.CreateLogStreamResponse{
				LinesInserted: logsInserted,
			})
			return nil
		} else if err != nil {
			return err
		} else if msg == nil {
			return status.Error(codes.InvalidArgument, "received nil message")
		}
		dbLogs := make([]database.Log, 0, len(msg.Lines))
		for _, line := range msg.Lines {
			if err := line.Timestamp.CheckValid(); err != nil {
				log.Warn().WithError(err).
					Message("Received invalid log timestamp, skipping.")
				continue
			}
			if line.BuildId == 0 {
				log.Warn().Message("Received log with build ID: 0, skipping.")
				continue
			}
			if line.BuildId > math.MaxUint {
				log.Warn().WithUint64("buildId", line.BuildId).
					Message("Received too big log build ID, skipping.")
				return status.Errorf(codes.InvalidArgument,
					"received build ID is too big: %d (build ID) > %d (max)",
					line.BuildId, uint(math.MaxUint))
			}
			dbLogs = append(dbLogs, database.Log{
				BuildID:   uint(line.BuildId),
				Message:   line.Message,
				Timestamp: line.Timestamp.AsTime(),
			})
		}
		log.Debug().WithInt("lines", len(msg.Lines)).Message("Received logs")
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
	}
}
