package controllers

import (
	"log"
	"os"
	tmpl "text/template"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	reformav1alpha1 "prosimcorp.com/reforma/api/v1alpha1"
	"github.com/Masterminds/sprig"
)

// CheckResource check the existence of a resource given by the Patch spec
func (r *PatchReconciler) CheckResource (reference corev1.ObjectReference) (err error) {
	return err
}

// GetSources return a list of Unstructured objects with the content of the sources
func (r *PatchReconciler) GetSources (*reformav1alpha1.Patch) (sources unstructured.UnstructuredList, err error) {
	return sources, err
}

// GetTarget return an unstructured object with the content of the target
func (r *PatchReconciler) GetTarget () (target unstructured.Unstructured, err error) {
	return target, err
}

// JoinResources return a list of unstructured objects appending the target resource to the sources list
func (r *PatchReconciler) JoinResources (sources unstructured.UnstructuredList, target unstructured.Unstructured) (resources unstructured.UnstructuredList, err error) {
	return resources, err
}

// ParseTemplate ... TODO
func (r *PatchReconciler) ParseTemplate (template string, resources unstructured.UnstructuredList) (err error) {

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