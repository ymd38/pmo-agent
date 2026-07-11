package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAggregateProjects(t *testing.T) {
	i64 := func(v int64) *int64 { return &v }
	day := func(d int) *time.Time {
		t := time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC)
		return &t
	}

	t.Run("空スライスは0件・予算0・日付nil・空StatusCounts", func(t *testing.T) {
		got := AggregateProjects(nil)
		assert.Equal(t, 0, got.ProjectCount)
		assert.Equal(t, int64(0), got.TotalBudget)
		assert.Nil(t, got.StartDate)
		assert.Nil(t, got.EndDate)
		assert.Equal(t, map[string]int{}, got.StatusCounts)
	})

	t.Run("予算はnilを0扱いで合算・期間はmin/max・ステータス別に集計", func(t *testing.T) {
		projects := []Project{
			{Budget: i64(100), StartDate: day(10), EndDate: day(20), Status: StatusActive},
			{Budget: nil, StartDate: day(5), EndDate: day(15), Status: StatusActive},
			{Budget: i64(50), StartDate: day(8), EndDate: day(30), Status: StatusPlanning},
			{Budget: i64(25), StartDate: nil, EndDate: nil, Status: StatusCompleted},
		}

		got := AggregateProjects(projects)

		assert.Equal(t, 4, got.ProjectCount)
		assert.Equal(t, int64(175), got.TotalBudget)
		assert.Equal(t, day(5).Unix(), got.StartDate.Unix(), "最小 start_date")
		assert.Equal(t, day(30).Unix(), got.EndDate.Unix(), "最大 end_date")
		assert.Equal(t, map[string]int{
			string(StatusActive):    2,
			string(StatusPlanning):  1,
			string(StatusCompleted): 1,
		}, got.StatusCounts)
	})
}
