package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseBackupStatusCondition is an enum of available status
// conditions for the DatabaseBackupStatus
type DatabaseBackupStatusCondition string

const (
	// ConditionSecretExists represents the status condition whether the secret was created successfully
	ConditionSecretExists DatabaseBackupStatusCondition = "db-backup.nect.com/secretExists" //#nosec:G101 -- That's not a credential
	// ConditionServiceExists represents the status condition whether the service was created successfully
	ConditionServiceExists DatabaseBackupStatusCondition = "db-backup.nect.com/serviceExists"
	// ConditionSTSExists represents the status condition whether the sts was created successfully
	ConditionSTSExists DatabaseBackupStatusCondition = "db-backup.nect.com/stsExists"

	conditionReady DatabaseBackupStatusCondition = "Ready"
)

var conditionSuccessStates = map[DatabaseBackupStatusCondition]metav1.ConditionStatus{
	ConditionSecretExists:  metav1.ConditionTrue,
	ConditionServiceExists: metav1.ConditionTrue,
	ConditionSTSExists:     metav1.ConditionTrue,
}

// Init initializes a new "not ready" status object
func (d *DatabaseBackupStatus) Init(generation int64) {
	for _, c := range []DatabaseBackupStatusCondition{
		ConditionSecretExists,
		ConditionServiceExists,
		ConditionSTSExists,
	} {
		d.set(c, generation, metav1.ConditionFalse, "init", "Object status initialized")
	}

	d.CalculateReady(generation)
}

// Set replaces the status condition if the status changed and
// re-calculates the overall Ready status
func (d *DatabaseBackupStatus) Set(condType DatabaseBackupStatusCondition, generation int64, status metav1.ConditionStatus, reason, message string) {
	d.set(condType, generation, status, reason, message)
	d.CalculateReady(generation)
}

// CalculateReady takes all known conditions into account and checks
// whether they are fine. From all those conditions the overall
// "Ready" condition is set
func (d *DatabaseBackupStatus) CalculateReady(generation int64) {
	var (
		message     = "Everything looks fine"
		readyStatus = metav1.ConditionTrue
	)
	for _, cond := range d.Conditions {
		expectedSuccess := conditionSuccessStates[DatabaseBackupStatusCondition(cond.Type)]
		if expectedSuccess == "" {
			// That's none of ours, lets skip it
			continue
		}

		switch {
		case cond.Status == metav1.ConditionUnknown && readyStatus == metav1.ConditionTrue:
			// If any component is unknown the overall status cannot be "True"
			message = "One or more conditions are in unknown state"
			readyStatus = metav1.ConditionUnknown

		case cond.Status == metav1.ConditionUnknown:
			// condition is unknown but readyStatus is already False / Unknown: Don't change!

		case cond.Status != expectedSuccess:
			// We have at least one non-fine status
			message = "One or more conditions are not fine"
			readyStatus = metav1.ConditionFalse
		}
	}

	d.set(conditionReady, generation, readyStatus, "calculateReady", message)
}

//revive:disable-next-line:confusing-naming -- That's intentional
func (d *DatabaseBackupStatus) set(condType DatabaseBackupStatusCondition, generation int64, status metav1.ConditionStatus, reason, message string) {
	var (
		found bool
		nc    = metav1.Condition{
			Type:               string(condType),
			Status:             status,
			ObservedGeneration: generation,
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		}
		tmp []metav1.Condition
	)

	for _, cond := range d.Conditions {
		if DatabaseBackupStatusCondition(cond.Type) != condType {
			// Not the condition we want to update, we keep that one
			tmp = append(tmp, cond)
			continue
		}

		if cond.Status == nc.Status && cond.Reason == nc.Reason && cond.Message == nc.Message && cond.ObservedGeneration == nc.ObservedGeneration {
			// nothing changed, we don't do anything
			return
		}

		tmp = append(tmp, nc)
		found = true
	}

	if !found {
		tmp = append(tmp, nc)
	}

	d.Conditions = tmp
}
