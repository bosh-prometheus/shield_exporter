package filters_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-community/shield_exporter/filters"
)

var _ = Describe("CollectorsFilter", func() {
	var (
		err     error
		filters []string

		collectorsFilter *CollectorsFilter
	)

	JustBeforeEach(func() {
		collectorsFilter, err = NewCollectorsFilter(filters)
	})

	Describe("New", func() {
		Context("when filters are supported", func() {
			BeforeEach(func() {
				filters = []string{
					ArchivesCollector,
					JobsCollector,
					RetentionPoliciesCollector,
					SchedulesCollector,
					StatusCollector,
					StoresCollector,
					TargetsCollector,
					TasksCollector,
				}
			})

			It("does not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when filters are not supported", func() {
			BeforeEach(func() {
				filters = []string{ArchivesCollector, "Unknown"}
			})

			It("returns an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Collector filter `Unknown` is not supported"))
			})
		})

		Context("when a filter has leading and/or trailing whitespaces", func() {
			BeforeEach(func() {
				filters = []string{"   " + ArchivesCollector + "  "}
			})

			It("returns an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Enabled", func() {
		Context("when collector is enabled", func() {
			BeforeEach(func() {
				filters = []string{ArchivesCollector, JobsCollector, RetentionPoliciesCollector, SchedulesCollector, StatusCollector, StoresCollector, TargetsCollector, TasksCollector}
			})

			It("Archives collector returns true", func() {
				Expect(collectorsFilter.Enabled(ArchivesCollector)).To(BeTrue())
			})

			It("Jobs collector returns true", func() {
				Expect(collectorsFilter.Enabled(JobsCollector)).To(BeTrue())
			})

			It("Retention Policies collector returns true", func() {
				Expect(collectorsFilter.Enabled(RetentionPoliciesCollector)).To(BeTrue())
			})

			It("Schedules collector returns true", func() {
				Expect(collectorsFilter.Enabled(SchedulesCollector)).To(BeTrue())
			})

			It("Status collector returns true", func() {
				Expect(collectorsFilter.Enabled(StatusCollector)).To(BeTrue())
			})

			It("Stores collector returns true", func() {
				Expect(collectorsFilter.Enabled(StoresCollector)).To(BeTrue())
			})

			It("Targets collector returns true", func() {
				Expect(collectorsFilter.Enabled(TargetsCollector)).To(BeTrue())
			})

			It("Tasks collector returns true", func() {
				Expect(collectorsFilter.Enabled(TasksCollector)).To(BeTrue())
			})
		})

		Context("when collector is not enabled", func() {
			BeforeEach(func() {
				filters = []string{ArchivesCollector}
			})

			It("Jobs collector returns false", func() {
				Expect(collectorsFilter.Enabled(JobsCollector)).To(BeFalse())
			})

			It("Retention Policies collector returns false", func() {
				Expect(collectorsFilter.Enabled(RetentionPoliciesCollector)).To(BeFalse())
			})

			It("Schedules collector returns false", func() {
				Expect(collectorsFilter.Enabled(SchedulesCollector)).To(BeFalse())
			})

			It("Status collector returns false", func() {
				Expect(collectorsFilter.Enabled(StatusCollector)).To(BeFalse())
			})

			It("Stores collector returns false", func() {
				Expect(collectorsFilter.Enabled(StoresCollector)).To(BeFalse())
			})

			It("Targets collector returns false", func() {
				Expect(collectorsFilter.Enabled(TargetsCollector)).To(BeFalse())
			})

			It("Tasks collector returns false", func() {
				Expect(collectorsFilter.Enabled(TasksCollector)).To(BeFalse())
			})
		})

		Context("when there are no filters", func() {
			BeforeEach(func() {
				filters = []string{}
			})

			It("Archives collector returns true", func() {
				Expect(collectorsFilter.Enabled(ArchivesCollector)).To(BeTrue())
			})

			It("Jobs collector returns true", func() {
				Expect(collectorsFilter.Enabled(JobsCollector)).To(BeTrue())
			})

			It("Retention Policies collector returns true", func() {
				Expect(collectorsFilter.Enabled(RetentionPoliciesCollector)).To(BeTrue())
			})

			It("Schedules collector returns true", func() {
				Expect(collectorsFilter.Enabled(SchedulesCollector)).To(BeTrue())
			})

			It("Status collector returns true", func() {
				Expect(collectorsFilter.Enabled(StatusCollector)).To(BeTrue())
			})

			It("Stores collector returns true", func() {
				Expect(collectorsFilter.Enabled(StoresCollector)).To(BeTrue())
			})

			It("Targets collector returns true", func() {
				Expect(collectorsFilter.Enabled(TargetsCollector)).To(BeTrue())
			})

			It("Tasks collector returns true", func() {
				Expect(collectorsFilter.Enabled(TasksCollector)).To(BeTrue())
			})
		})
	})
})
