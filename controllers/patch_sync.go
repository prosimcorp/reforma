package controllers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"log"
	reformav1alpha1 "prosimcorp.com/reforma/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	tmpl "text/template"
)

const (
	// ErrorInvalidPatchTypeMessage error message for invalid values on 'patchType' parameter
	ErrorInvalidPatchTypeMessage = "PatchType: invalid value. Choose one of the following: %s"
)

var (
	// AvailabePatchTypes store a list with all available values for patchTypes parameter
	AvailabePatchTypes = []string{
		string(types.JSONPatchType),
		string(types.MergePatchType),
		string(types.StrategicMergePatchType),
		string(types.ApplyPatchType),
	}
)

// GetSources return a pointer to a list of Unstructured objects with the content of the sources
func (r *PatchReconciler) GetSources(ctx context.Context, patch *reformav1alpha1.Patch) (sources *unstructured.UnstructuredList, err error) {

	sources = &unstructured.UnstructuredList{}

	// Get the source content, one by one
	sourceObject := &unstructured.Unstructured{}
	for _, sourceReference := range patch.Spec.Sources {
		sourceObject.SetGroupVersionKind(sourceReference.GroupVersionKind())

		err = r.Get(ctx, client.ObjectKey{
			Namespace: sourceReference.Namespace,
			Name:      sourceReference.Name,
		}, sourceObject)

		if err != nil {
			return sources, err
		}

		sources.Items = append(sources.Items, *sourceObject)
	}

	return sources, err
}

// GetTarget returns a pointer to an Unstructured object with the content of the target
func (r *PatchReconciler) GetTarget(ctx context.Context, patch *reformav1alpha1.Patch) (target *unstructured.Unstructured, err error) {

	// Get the target manifest
	target = &unstructured.Unstructured{}
	target.SetGroupVersionKind(patch.Spec.Target.GroupVersionKind())

	err = r.Get(ctx, client.ObjectKey{
		Namespace: patch.Spec.Target.Namespace,
		Name:      patch.Spec.Target.Name,
	}, target)

	return target, err
}

// GetResources return an UnstructuredList whose items are the target and the sources
func (r *PatchReconciler) getResources(ctx context.Context, patch *reformav1alpha1.Patch) (resources *unstructured.UnstructuredList, err error) {

	resources = &unstructured.UnstructuredList{}

	sources, err := r.GetSources(ctx, patch)
	if err != nil {
		// TODO: Update the CR status
		return resources, err
	}

	target, err := r.GetTarget(ctx, patch)
	if err != nil {
		// TODO: Update the CR status
		return resources, err
	}

	resources.Items = append(resources.Items, *target)
	resources.Items = append(resources.Items, sources.Items...)

	return resources, err
}

// GetResources return a JSON compatible list of objects with the target and the sources
func (r *PatchReconciler) GetResources(ctx context.Context, patch *reformav1alpha1.Patch) (parsedResources []map[string]interface{}, err error) {

	resources, err := r.getResources(ctx, patch)

	if err != nil {
		// TODO: Update the CR status
		return parsedResources, err
	}

	// Transform the UnstructuredList into a list of JSON compatible objects
	for _, resource := range resources.Items {
		parsedResources = append(parsedResources, resource.Object)
	}

	return parsedResources, err
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

// GetPatchType return the patchType string from a Patch CR
func (r *PatchReconciler) GetPatchType(ctx context.Context, patch *reformav1alpha1.Patch) (patchType string, err error) {

	patchType = string(patch.Spec.PatchType)

	for _, AvailabePatchType := range AvailabePatchTypes {
		if patchType == AvailabePatchType {
			return patchType, nil
		}
	}

	// TODO Change the status conditions
	return patchType, fmt.Errorf(ErrorInvalidPatchTypeMessage, strings.Join(AvailabePatchTypes, ", "))
}

// GetPatch return the patch string already prepared to call the Kubernetes API
func (r *PatchReconciler) GetPatch(ctx context.Context, patch *reformav1alpha1.Patch) (parsedPatch string, err error) {

	// Map useful sprig functions to give superpower to the users
	templateFunctionsMap := r.GetFunctionsMap()

	// Get the resources from a Patch CR
	resources, err := r.GetResources(ctx, patch)
	if err != nil {
		// TODO Change the status conditions here or inside the child
		return parsedPatch, err
	}

	// Create a Template object from the given string
	template, err := tmpl.New("main").Funcs(templateFunctionsMap).Parse(patch.Spec.Template)
	if err != nil {
		// TODO Change the status conditions
		return parsedPatch, err
	}

	// Create a new buffer to store the templating result
	buffer := new(bytes.Buffer)

	err = template.Execute(buffer, resources)
	if err != nil {
		// TODO Change the status conditions
		return parsedPatch, err
	}

	parsedPatch = buffer.String()
	return parsedPatch, err
}

// PatchTarget call Kubernetes API to actually patch the resource
func (r *PatchReconciler) PatchTarget(ctx context.Context, patch *reformav1alpha1.Patch) (err error) {

	//sources, err := r.GetSources(ctx, patch)
	//
	//if err != nil {
	//	log.Print("Algo ha pasado con los sources, pavo")
	//}
	//
	//log.Print("SOURCES-----------------------")
	//log.Print(sources.Items)
	//log.Print("------------------------------")
	//
	//target, err := r.GetTarget(ctx, patch)
	//
	//if err != nil {
	//	log.Print("Algo ha pasado con el target, pavo")
	//}
	//
	//log.Print("TARGET-----------------------")
	//log.Print(target)
	//log.Print("------------------------------")

	patchType, err := r.GetPatchType(ctx, patch)

	if err != nil {
		log.Print("GetPatchType ------------------------------")
		log.Print(patchType)
		log.Print(err)
		log.Print("GetPatchType END ------------------------------")
	}

	parsedPatch, err := r.GetPatch(ctx, patch)

	if err != nil {
		log.Print("GetPatch ------------------------------")
		log.Print(err)
		log.Print("GetPatch END ------------------------------")
	}

	log.Print("PATCH-----------------------")
	log.Print(parsedPatch)
	log.Print("------------------------------")

	return err
}
