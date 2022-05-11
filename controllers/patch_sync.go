package controllers

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	tmpl "text/template"

	reformav1alpha1 "prosimcorp.com/reforma/api/v1alpha1"


	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yamlsigs "sigs.k8s.io/yaml"
)

const (
	// ErrorInvalidPatchTypeMessage error message for invalid values on 'patchType' parameter
	ErrorInvalidPatchTypeMessage = "PatchType: invalid value. Choose one of the following: %s"
)

var (
	// AvailabePatchTypes store a list with all available values for patchTypes parameter
	AvailabePatchTypes = []types.PatchType{
		types.JSONPatchType,
		types.MergePatchType,
		types.StrategicMergePatchType,
		types.ApplyPatchType,
	}
)

// GetPatchTypesString return a list of all available patch types as strings for later convenience
func GetPatchTypesString() (types []string) {
	for _, str := range AvailabePatchTypes {
		types = append(types, string(str))
	}
	return types
}

// addSources fill the resources list from input parameters with the content of the sources
func (r *PatchReconciler) addSources(ctx context.Context, patchManifest *reformav1alpha1.Patch, resources *[]map[string]interface{}) (err error) {

	// Fill the sources content, one by one
	sourceObject := &unstructured.Unstructured{}

	for _, sourceReference := range patchManifest.Spec.Sources {
		sourceObject.SetGroupVersionKind(sourceReference.GroupVersionKind())

		err = r.Get(ctx, client.ObjectKey{
			Namespace: sourceReference.Namespace,
			Name:      sourceReference.Name,
		}, sourceObject)

		if err != nil {
			return err
		}

		*resources = append(*resources, sourceObject.Object)
	}

	return err
}

// addTarget fill the resources list from input parameters with the target object content
func (r *PatchReconciler) addTarget(ctx context.Context, patchManifest *reformav1alpha1.Patch, resources *[]map[string]interface{}) (err error) {

	// Get the target manifest
	target := &unstructured.Unstructured{}
	target.SetGroupVersionKind(patchManifest.Spec.Target.GroupVersionKind())

	err = r.Get(ctx, client.ObjectKey{
		Namespace: patchManifest.Spec.Target.Namespace,
		Name:      patchManifest.Spec.Target.Name,
	}, target)
	if err != nil {
		return err
	}

	*resources = append(*resources, target.Object)

	return err
}

// GetResources return a JSON compatible list of objects with the target and the sources
func (r *PatchReconciler) GetResources(ctx context.Context, patchManifest *reformav1alpha1.Patch) (resources []map[string]interface{}, err error) {

	// Fill the resources list with the target
	err = r.addTarget(ctx, patchManifest, &resources)
	if err != nil {
		r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeResourcePatched,
			metav1.ConditionFalse,
			ConditionReasonTargetNotFound,
			ConditionReasonTargetNotFoundMessage,
		))
	}

	// Fill the resources list with the sources
	err = r.addSources(ctx, patchManifest, &resources)
	if err != nil {
		r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeResourcePatched,
			metav1.ConditionFalse,
			ConditionReasonSourceNotFound,
			ConditionReasonSourceNotFoundMessage,
		))
	}

	return resources, err
}

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string)
// Ref: https://github.com/helm/helm/blob/main/pkg/engine/funcs.go#L79-L90
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// fromYAML converts a YAML document into a map[string]interface{}
// Ref: https://github.com/helm/helm/blob/main/pkg/engine/funcs.go#L92-L105
func fromYAML(str string) map[string]interface{} {
	m := map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(str), &m); err != nil {
		m["Error"] = err.Error()
	}
	return m
}

// GetFunctionsMap return a map with equivalency between functions for inside templating and real Golang ones
func (r *PatchReconciler) GetFunctionsMap() tmpl.FuncMap {
	f := sprig.TxtFuncMap()
	f["toYaml"] = toYAML
	f["fromYaml"] = fromYAML
	return f
}

// CheckPatchType check if the 'patchType' in the Path CR is available
func (r *PatchReconciler) CheckPatchType(patchManifest *reformav1alpha1.Patch) (err error) {

	for _, AvailabePatchType := range AvailabePatchTypes {
		if AvailabePatchType == patchManifest.Spec.PatchType {
			return err
		}
	}

	r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeResourcePatched,
		metav1.ConditionFalse,
		ConditionReasonInvalidPatchType,
		ConditionReasonInvalidPatchTypeMessage,
	))
	err = fmt.Errorf(ErrorInvalidPatchTypeMessage, strings.Join(GetPatchTypesString(), ", "))

	return err
}

// GetPatch return the patch string already prepared to call the Kubernetes API
func (r *PatchReconciler) GetPatch(ctx context.Context, patchManifest *reformav1alpha1.Patch) (parsedPatch string, err error) {

	// Map useful sprig functions to give superpower to the users
	templateFunctionsMap := r.GetFunctionsMap()

	// Get the resources from a Patch CR
	resources, err := r.GetResources(ctx, patchManifest)
	if err != nil {
		return parsedPatch, err
	}

	// Create a Template object from the given string
	template, err := tmpl.New("main").Funcs(templateFunctionsMap).Parse(patchManifest.Spec.Template)
	if err != nil {
		r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeTemplateSucceed,
			metav1.ConditionFalse,
			ConditionReasonTemplateParsingFailed,
			fmt.Sprintf(ConditionReasonTemplateParsingFailedMessage, err.Error()),
		))
		return parsedPatch, err
	}

	// Create a new buffer to store the templating result
	buffer := new(bytes.Buffer)

	err = template.Execute(buffer, resources)
	if err != nil {
		r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeTemplateSucceed,
			metav1.ConditionFalse,
			ConditionReasonTemplateExecutionFailed,
			fmt.Sprintf(ConditionReasonTemplateExecutionFailedMessage, err.Error()),
		))
		return parsedPatch, err
	}

	parsedPatch = buffer.String()

	r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeTemplateSucceed,
		metav1.ConditionTrue,
		ConditionReasonTemplateParsed,
		ConditionReasonTemplateParsedMessage,
	))

	return parsedPatch, err
}

// PatchTarget call Kubernetes API to actually patch the resource
func (r *PatchReconciler) PatchTarget(ctx context.Context, patchManifest *reformav1alpha1.Patch) (err error) {

	err = r.CheckPatchType(patchManifest)
	if err != nil {
		r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeResourcePatched,
			metav1.ConditionFalse,
			ConditionReasonInvalidPatchType,
			ConditionReasonInvalidPatchTypeMessage,
		))
		return err
	}

	patch, err := r.GetPatch(ctx, patchManifest)
	if err != nil {
		return err
	}

	// Get the target to patch
	target := &unstructured.Unstructured{}
	target.SetGroupVersionKind(patchManifest.Spec.Target.GroupVersionKind())
	err = r.Get(ctx, client.ObjectKey {
		Namespace: patchManifest.Spec.Target.Namespace,
		Name:      patchManifest.Spec.Target.Name,
	}, target)
	if err != nil {
		return err
	}

	// Convert the YAML patch to JSON because, remember, Kubernetes use JSON internally
	patchJSON, err := yamlsigs.YAMLToJSON([]byte(patch))
	if err != nil {
		return err
	}

	// Actually perform the patch against Kubernetes
	err = r.Patch(ctx, target, client.RawPatch(patchManifest.Spec.PatchType, patchJSON))
	if err != nil {
		return err
	}

	log.Print("PatchTarget END ------------------------------")

	return err
}
