package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	reformav1alpha1 "prosimcorp.com/reforma/api/v1alpha1"
)

// https://github.com/external-secrets/external-secrets/blob/80545f4f183795ef193747fc959558c761b51c99/apis/externalsecrets/v1alpha1/externalsecret_types.go#L168
const (
	// ConditionTypeResourcePatched indicates that the target was patched or not
	ConditionTypeResourcePatched = "ResourcePatched"

	// Source not found
	ConditionReasonSourceNotFound        = "SourceNotFound"
	ConditionReasonSourceNotFoundMessage = "Source resource was not found"

	// Target not found
	ConditionReasonTargetNotFound        = "TargetNotFound"
	ConditionReasonTargetNotFoundMessage = "Target resource was not found"

	// Invalid patch type
	ConditionReasonInvalidPatchType        = "InvalidPatchType"
	ConditionReasonInvalidPatchTypeMessage = "Patch type is not supported"

	// Template failed
	ConditionReasonInvalidTemplate        = "InvalidTemplate"
	ConditionReasonInvalidTemplateMessage = "Patch template is not valid. Deeper information inside the Patch status"

	// Failure
	ConditionReasonInvalidPatch        = "InvalidPatch"
	ConditionReasonInvalidPatchMessage = "Patch is invalid"

	// Success
	ConditionReasonTargetPatched        = "TargetPatched"
	ConditionReasonTargetPatchedMessage = "Target was successfully patched"

	// ConditionTypeTemplateSucceed indicates that the templating stage was performed successfully
	ConditionTypeTemplateSucceed = "TemplateSucceed"

	// Template parsing failed
	ConditionReasonTemplateParsingFailed        = "TemplateParsingFailed"
	ConditionReasonTemplateParsingFailedMessage = "Golang returned: %s"

	// Template execution failed
	ConditionReasonTemplateExecutionFailed        = "TemplateExecutionFailed"
	ConditionReasonTemplateExecutionFailedMessage = "Golang returned: %s"

	// Success
	ConditionReasonTemplateParsed        = "TemplateParsed"
	ConditionReasonTemplateParsedMessage = "Template was successfully parsed"
)

// NewPatchCondition a set of default options for creating a Condition.
func (r *PatchReconciler) NewPatchCondition(condType string, status metav1.ConditionStatus, reason, message string) *metav1.Condition {
	return &metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetPatchCondition returns the condition with the provided type.
func (r *PatchReconciler) GetPatchCondition(patch *reformav1alpha1.Patch, condType string) *metav1.Condition {

	for i, v := range patch.Status.Conditions {
		if v.Type == condType {
			return &patch.Status.Conditions[i]
		}
	}
	return nil
}

// UpdatePatchCondition update or create a new condition inside the status of the CR
func (r *PatchReconciler) UpdatePatchCondition(patch *reformav1alpha1.Patch, condition *metav1.Condition) {

	// Get the condition
	currentCondition := r.GetPatchCondition(patch, condition.Type)

	if currentCondition == nil {
		// Create the condition when not existent
		patch.Status.Conditions = append(patch.Status.Conditions, *condition)
	} else {
		// Update the condition when existent.
		currentCondition.Status = condition.Status
		currentCondition.Reason = condition.Reason
		currentCondition.Message = condition.Message
		currentCondition.LastTransitionTime = metav1.Now()
	}
}
