package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	wfpb "github.com/dwrui/go-zero-admin-pkg/rpcclient/workflow/workflow"
	"google.golang.org/grpc"
)

const (
	AuditPending    int64 = 0
	AuditInProgress int64 = 1
	AuditApproved   int64 = 2
	AuditRejected   int64 = 3
)

// InstanceStarter 启动审批流程所需的最小 gRPC 接口。
type InstanceStarter interface {
	StartInstance(ctx context.Context, in *wfpb.StartInstanceRequest, opts ...grpc.CallOption) (*wfpb.StartInstanceResponse, error)
}

// ValidateAuditStatus 校验业务单据当前审核状态是否允许再次提交。
func ValidateAuditStatus(status int64) error {
	switch status {
	case AuditInProgress:
		return fmt.Errorf("审批进行中，请勿重复提交")
	case AuditApproved:
		return fmt.Errorf("审批已通过，不可重复提交")
	}
	return nil
}

// SubmitParams 提交审批的通用参数。
type SubmitParams struct {
	ProcessKey    string
	BizType       string
	BizId         uint64
	BizNo         string
	Title         string
	FormData      map[string]interface{}
	StartUserId   uint64
	StartUserName string
	StartDeptId   uint64
	BusinessId    int64
	DeptId        int64
	CreateBy      int64
}

func (p SubmitParams) toRequest() *wfpb.StartInstanceRequest {
	formJSON := "{}"
	if len(p.FormData) > 0 {
		if b, err := json.Marshal(p.FormData); err == nil {
			formJSON = string(b)
		}
	}
	return &wfpb.StartInstanceRequest{
		ProcessKey:    p.ProcessKey,
		BizType:       p.BizType,
		BizId:         p.BizId,
		BizNo:         p.BizNo,
		Title:         p.Title,
		FormData:      formJSON,
		StartUserId:   p.StartUserId,
		StartUserName: p.StartUserName,
		StartDeptId:   p.StartDeptId,
		BusinessId:    p.BusinessId,
		DeptId:        p.DeptId,
		CreateBy:      p.CreateBy,
	}
}

// StartAudit 启动审批流程；失败时执行 onFail 回滚（如将 audit_status 置为已拒绝）。
func StartAudit(ctx context.Context, client InstanceStarter, params SubmitParams, onFail func() error) (uint64, error) {
	resp, err := client.StartInstance(ctx, params.toRequest())
	if err != nil {
		if onFail != nil {
			_ = onFail()
		}
		return 0, fmt.Errorf("启动审批流程失败: %v", err)
	}
	if resp == nil {
		return 0, fmt.Errorf("启动审批流程失败: 空响应")
	}
	return resp.InstanceId, nil
}

// FormatBizNo 生成业务单据编号，如 WH12、WP34。
func FormatBizNo(prefix string, id uint64) string {
	return fmt.Sprintf("%s%d", prefix, id)
}
