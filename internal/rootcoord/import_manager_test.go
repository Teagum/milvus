// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rootcoord

import (
	"context"
	"testing"

	"github.com/milvus-io/milvus/internal/proto/commonpb"
	"github.com/milvus-io/milvus/internal/proto/datapb"
	"github.com/milvus-io/milvus/internal/proto/milvuspb"
	"github.com/milvus-io/milvus/internal/proto/rootcoordpb"
	"github.com/stretchr/testify/assert"
)

func TestImportManager_NewImportManager(t *testing.T) {
	fn := func(ctx context.Context, req *datapb.ImportTask) *commonpb.Status {
		return &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		}
	}
	mgr := newImportManager(context.TODO(), nil, fn)
	assert.NotNil(t, mgr)
}

func TestImportManager_ImportJob(t *testing.T) {
	mgr := newImportManager(context.TODO(), nil, nil)
	resp := mgr.importJob(nil)
	assert.NotEqual(t, commonpb.ErrorCode_Success, resp.Status.ErrorCode)

	rowReq := &milvuspb.ImportRequest{
		CollectionName: "c1",
		PartitionName:  "p1",
		RowBased:       true,
		Files:          []string{"f1", "f2", "f3"},
	}

	resp = mgr.importJob(rowReq)
	assert.NotEqual(t, commonpb.ErrorCode_Success, resp.Status.ErrorCode)

	colReq := &milvuspb.ImportRequest{
		CollectionName: "c1",
		PartitionName:  "p1",
		RowBased:       false,
		Files:          []string{"f1", "f2"},
		Options: []*commonpb.KeyValuePair{
			{
				Key:   Bucket,
				Value: "mybucket",
			},
		},
	}

	fn := func(ctx context.Context, req *datapb.ImportTask) *commonpb.Status {
		return &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_UnexpectedError,
		}
	}

	mgr = newImportManager(context.TODO(), nil, fn)
	resp = mgr.importJob(rowReq)
	assert.Equal(t, len(rowReq.Files), len(mgr.pendingTasks))
	assert.Equal(t, 0, len(mgr.workingTasks))

	mgr = newImportManager(context.TODO(), nil, fn)
	resp = mgr.importJob(colReq)
	assert.Equal(t, 1, len(mgr.pendingTasks))
	assert.Equal(t, 0, len(mgr.workingTasks))

	fn = func(ctx context.Context, req *datapb.ImportTask) *commonpb.Status {
		return &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		}
	}

	mgr = newImportManager(context.TODO(), nil, fn)
	resp = mgr.importJob(rowReq)
	assert.Equal(t, 0, len(mgr.pendingTasks))
	assert.Equal(t, len(rowReq.Files), len(mgr.workingTasks))

	mgr = newImportManager(context.TODO(), nil, fn)
	resp = mgr.importJob(colReq)
	assert.Equal(t, 0, len(mgr.pendingTasks))
	assert.Equal(t, 1, len(mgr.workingTasks))

	count := 0
	fn = func(ctx context.Context, req *datapb.ImportTask) *commonpb.Status {
		if count >= 2 {
			return &commonpb.Status{
				ErrorCode: commonpb.ErrorCode_UnexpectedError,
			}
		}
		count++
		return &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		}
	}

	mgr = newImportManager(context.TODO(), nil, fn)
	resp = mgr.importJob(rowReq)
	assert.Equal(t, len(rowReq.Files)-2, len(mgr.pendingTasks))
	assert.Equal(t, 2, len(mgr.workingTasks))
}

func TestImportManager_TaskState(t *testing.T) {
	fn := func(ctx context.Context, req *datapb.ImportTask) *commonpb.Status {
		return &commonpb.Status{
			ErrorCode: commonpb.ErrorCode_Success,
		}
	}

	rowReq := &milvuspb.ImportRequest{
		CollectionName: "c1",
		PartitionName:  "p1",
		RowBased:       true,
		Files:          []string{"f1", "f2", "f3"},
	}

	mgr := newImportManager(context.TODO(), nil, fn)
	mgr.importJob(rowReq)

	state := &rootcoordpb.ImportResult{
		TaskId: 10000,
	}
	err := mgr.updateTaskState(state)
	assert.NotNil(t, err)

	state = &rootcoordpb.ImportResult{
		TaskId:   1,
		RowCount: 1000,
		State:    commonpb.ImportState_ImportCompleted,
	}
	err = mgr.updateTaskState(state)
	assert.Nil(t, err)

	resp := mgr.getTaskState(10000)
	assert.Equal(t, commonpb.ErrorCode_UnexpectedError, resp.Status.ErrorCode)

	resp = mgr.getTaskState(1)
	assert.Equal(t, commonpb.ErrorCode_Success, resp.Status.ErrorCode)
	assert.Equal(t, commonpb.ImportState_ImportCompleted, resp.State)

	resp = mgr.getTaskState(0)
	assert.Equal(t, commonpb.ErrorCode_Success, resp.Status.ErrorCode)
	assert.Equal(t, commonpb.ImportState_ImportPending, resp.State)
}
