package controllers

import (
	"context"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	tmpl "text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	reformav1alpha1 "prosimcorp.com/reforma/api/v1alpha1"
)

// GetSources return a pointer to a list of Unstructured objects with the content of the sources
func (r *PatchReconciler) GetSources(ctx context.Context, patch *reformav1alpha1.Patch) (sources *unstructured.UnstructuredList, err error) {

	sources = &unstructured.UnstructuredList{}

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

// JoinResources return a list of unstructured objects appending the target resource to the sources list
func (r *PatchReconciler) JoinResources(sources unstructured.UnstructuredList, target unstructured.Unstructured) (resources unstructured.UnstructuredList, err error) {
	return resources, err
}

// ParseTemplate ... TODO
func (r *PatchReconciler) ParseTemplate(template string, resources unstructured.UnstructuredList) (err error) {

	sprigFuncs := sprig.TxtFuncMap()

	masterTmpl, err := tmpl.New("master").Funcs(sprigFuncs).Parse(template)
	if err != nil {
		log.Fatal(err) // TODO
	}

	// Execute the template inserting the list of resources (sources and target) for substitution
	if err = masterTmpl.Execute(os.Stdout, resources.Items); err != nil {
		log.Fatal(err) // TODO
	}

	return err
}
