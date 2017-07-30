package filters

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ArchivesCollector          = "Archives"
	JobsCollector              = "Jobs"
	RetentionPoliciesCollector = "RetentionPolicies"
	SchedulesCollector         = "Schedules"
	StatusCollector            = "Status"
	StoresCollector            = "Stores"
	TargetsCollector           = "Targets"
	TasksCollector             = "Tasks"
)

type CollectorsFilter struct {
	collectorsEnabled map[string]bool
}

func NewCollectorsFilter(filters []string) (*CollectorsFilter, error) {
	collectorsEnabled := make(map[string]bool)

	for _, collectorName := range filters {
		switch strings.Trim(collectorName, " ") {
		case ArchivesCollector:
			collectorsEnabled[ArchivesCollector] = true
		case JobsCollector:
			collectorsEnabled[JobsCollector] = true
		case RetentionPoliciesCollector:
			collectorsEnabled[RetentionPoliciesCollector] = true
		case SchedulesCollector:
			collectorsEnabled[SchedulesCollector] = true
		case StatusCollector:
			collectorsEnabled[StatusCollector] = true
		case StoresCollector:
			collectorsEnabled[StoresCollector] = true
		case TargetsCollector:
			collectorsEnabled[TargetsCollector] = true
		case TasksCollector:
			collectorsEnabled[TasksCollector] = true
		default:
			return &CollectorsFilter{}, errors.New(fmt.Sprintf("Collector filter `%s` is not supported", collectorName))
		}
	}

	return &CollectorsFilter{collectorsEnabled: collectorsEnabled}, nil
}

func (f *CollectorsFilter) Enabled(collectorName string) bool {
	if len(f.collectorsEnabled) == 0 {
		return true
	}

	if f.collectorsEnabled[collectorName] {
		return true
	}

	return false
}
