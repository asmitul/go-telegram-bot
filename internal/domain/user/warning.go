package user

import "time"

// Warning 警告记录
type Warning struct {
	ID        int64     // 警告ID
	UserID    int64     // 被警告用户ID
	GroupID   int64     // 群组ID
	Reason    string    // 警告原因
	IssuedBy  int64     // 发出警告的管理员ID
	IssuedAt  time.Time // 警告时间
	IsCleared bool      // 是否已清除
}

// NewWarning 创建新警告
func NewWarning(userID, groupID int64, reason string, issuedBy int64) *Warning {
	return &Warning{
		UserID:    userID,
		GroupID:   groupID,
		Reason:    reason,
		IssuedBy:  issuedBy,
		IssuedAt:  time.Now(),
		IsCleared: false,
	}
}

// Clear 清除警告
func (w *Warning) Clear() {
	w.IsCleared = true
}

// WarningRepository 警告仓储接口
type WarningRepository interface {
	// Save 保存警告
	Save(warning *Warning) error

	// FindByUserAndGroup 查找用户在特定群组的所有警告
	FindByUserAndGroup(userID, groupID int64) ([]*Warning, error)

	// CountActiveWarnings 统计用户在特定群组的有效警告数量
	CountActiveWarnings(userID, groupID int64) (int, error)

	// ClearWarnings 清除用户在特定群组的所有警告
	ClearWarnings(userID, groupID int64) error

	// Delete 删除警告
	Delete(id int64) error
}
