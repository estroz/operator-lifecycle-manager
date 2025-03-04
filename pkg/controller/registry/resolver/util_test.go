package resolver

import (
	"encoding/json"
	"testing"

	"github.com/operator-framework/operator-registry/pkg/api"
	opregistry "github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/resolver/cache"
)

// RequireStepsEqual is similar to require.ElementsMatch, but produces better error messages
func RequireStepsEqual(t *testing.T, expectedSteps, steps []*v1alpha1.Step) {
	for _, s := range expectedSteps {
		require.Contains(t, steps, s, "step in expected not found in steps")
	}
	for _, s := range steps {
		require.Contains(t, expectedSteps, s, "step in steps not found in expected")
	}
}

func csv(name, replaces string, ownedCRDs, requiredCRDs, ownedAPIServices, requiredAPIServices cache.APISet, permissions, clusterPermissions []v1alpha1.StrategyDeploymentPermissions) *v1alpha1.ClusterServiceVersion {
	var singleInstance = int32(1)
	strategy := v1alpha1.StrategyDetailsDeployment{
		Permissions:        permissions,
		ClusterPermissions: clusterPermissions,
		DeploymentSpecs: []v1alpha1.StrategyDeploymentSpec{
			{
				Name: name,
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": name,
						},
					},
					Replicas: &singleInstance,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": name,
							},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: "sa",
							Containers: []corev1.Container{
								{
									Name:  name + "-c1",
									Image: "nginx:1.7.9",
									Ports: []corev1.ContainerPort{
										{
											ContainerPort: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	installStrategy := v1alpha1.NamedInstallStrategy{
		StrategyName: v1alpha1.InstallStrategyNameDeployment,
		StrategySpec: strategy,
	}

	requiredCRDDescs := make([]v1alpha1.CRDDescription, 0)
	for crd := range requiredCRDs {
		requiredCRDDescs = append(requiredCRDDescs, v1alpha1.CRDDescription{Name: crd.Plural + "." + crd.Group, Version: crd.Version, Kind: crd.Kind})
	}

	ownedCRDDescs := make([]v1alpha1.CRDDescription, 0)
	for crd := range ownedCRDs {
		ownedCRDDescs = append(ownedCRDDescs, v1alpha1.CRDDescription{Name: crd.Plural + "." + crd.Group, Version: crd.Version, Kind: crd.Kind})
	}

	requiredAPIDescs := make([]v1alpha1.APIServiceDescription, 0)
	for api := range requiredAPIServices {
		requiredAPIDescs = append(requiredAPIDescs, v1alpha1.APIServiceDescription{Name: api.Plural, Group: api.Group, Version: api.Version, Kind: api.Kind})
	}

	ownedAPIDescs := make([]v1alpha1.APIServiceDescription, 0)
	for api := range ownedAPIServices {
		ownedAPIDescs = append(ownedAPIDescs, v1alpha1.APIServiceDescription{Name: api.Plural, Group: api.Group, Version: api.Version, Kind: api.Kind, DeploymentName: name, ContainerPort: 80})
	}

	return &v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.ClusterServiceVersionKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ClusterServiceVersionSpec{
			Replaces:        replaces,
			InstallStrategy: installStrategy,
			CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
				Owned:    ownedCRDDescs,
				Required: requiredCRDDescs,
			},
			APIServiceDefinitions: v1alpha1.APIServiceDefinitions{
				Owned:    ownedAPIDescs,
				Required: requiredAPIDescs,
			},
		},
	}
}

func crd(key opregistry.APIKey) *v1beta1.CustomResourceDefinition {
	return &v1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: key.Plural + "." + key.Group,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: key.Group,
			Versions: []v1beta1.CustomResourceDefinitionVersion{
				{
					Name:    key.Version,
					Storage: true,
					Served:  true,
				},
			},
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   key.Kind,
				Plural: key.Plural,
			},
		},
	}
}

func u(object runtime.Object) *unstructured.Unstructured {
	unst, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: unst}
}

func apiSetToGVK(crds, apis cache.APISet) (out []*api.GroupVersionKind) {
	out = make([]*api.GroupVersionKind, 0)
	for a := range crds {
		out = append(out, &api.GroupVersionKind{
			Group:   a.Group,
			Version: a.Version,
			Kind:    a.Kind,
			Plural:  a.Plural,
		})
	}
	for a := range apis {
		out = append(out, &api.GroupVersionKind{
			Group:   a.Group,
			Version: a.Version,
			Kind:    a.Kind,
			Plural:  a.Plural,
		})
	}
	return
}

func packageNameToProperty(packageName, version string) (out *api.Property) {
	val, err := json.Marshal(opregistry.PackageProperty{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		panic(err)
	}

	return &api.Property{
		Type:  opregistry.PackageType,
		Value: string(val),
	}
}

type bundleOpt func(*api.Bundle)

func withSkipRange(skipRange string) bundleOpt {
	return func(b *api.Bundle) {
		b.SkipRange = skipRange
	}
}

func withSkips(skips []string) bundleOpt {
	return func(b *api.Bundle) {
		b.Skips = skips
	}
}

func withVersion(version string) bundleOpt {
	return func(b *api.Bundle) {
		b.Version = version
		props := b.GetProperties()
		for i, p := range props {
			if p.Type == opregistry.PackageType {
				props[i] = packageNameToProperty(b.PackageName, b.Version)
			}
		}
		b.Properties = props
	}
}

func bundle(name, pkg, channel, replaces string, providedCRDs, requiredCRDs, providedAPIServices, requiredAPIServices cache.APISet, opts ...bundleOpt) *api.Bundle {
	csvJson, err := json.Marshal(csv(name, replaces, providedCRDs, requiredCRDs, providedAPIServices, requiredAPIServices, nil, nil))
	if err != nil {
		panic(err)
	}

	objs := []string{string(csvJson)}
	for p := range providedCRDs {
		crdJson, err := json.Marshal(crd(p))
		if err != nil {
			panic(err)
		}
		objs = append(objs, string(crdJson))
	}

	b := &api.Bundle{
		CsvName:      name,
		PackageName:  pkg,
		ChannelName:  channel,
		CsvJson:      string(csvJson),
		Object:       objs,
		Version:      "0.0.0",
		ProvidedApis: apiSetToGVK(providedCRDs, providedAPIServices),
		RequiredApis: apiSetToGVK(requiredCRDs, requiredAPIServices),
		Replaces:     replaces,
		Dependencies: apiSetToDependencies(requiredCRDs, requiredAPIServices),
		Properties: append(apiSetToProperties(providedCRDs, providedAPIServices, false),
			packageNameToProperty(pkg, "0.0.0"),
		),
	}
	for _, f := range opts {
		f(b)
	}
	return b
}

func stripManifests(bundle *api.Bundle) *api.Bundle {
	bundle.CsvJson = ""
	bundle.Object = nil
	return bundle
}

func withBundleObject(bundle *api.Bundle, obj *unstructured.Unstructured) *api.Bundle {
	j, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	bundle.Object = append(bundle.Object, string(j))
	return bundle
}

func withBundlePath(bundle *api.Bundle, path string) *api.Bundle {
	bundle.BundlePath = path
	return bundle
}

func bundleWithPermissions(name, pkg, channel, replaces string, providedCRDs, requiredCRDs, providedAPIServices, requiredAPIServices cache.APISet, permissions, clusterPermissions []v1alpha1.StrategyDeploymentPermissions) *api.Bundle {
	csvJson, err := json.Marshal(csv(name, replaces, providedCRDs, requiredCRDs, providedAPIServices, requiredAPIServices, permissions, clusterPermissions))
	if err != nil {
		panic(err)
	}

	objs := []string{string(csvJson)}
	for p := range providedCRDs {
		crdJson, err := json.Marshal(crd(p))
		if err != nil {
			panic(err)
		}
		objs = append(objs, string(crdJson))
	}

	return &api.Bundle{
		CsvName:      name,
		PackageName:  pkg,
		ChannelName:  channel,
		CsvJson:      string(csvJson),
		Object:       objs,
		ProvidedApis: apiSetToGVK(providedCRDs, providedAPIServices),
		RequiredApis: apiSetToGVK(requiredCRDs, requiredAPIServices),
	}
}

func withReplaces(operator *cache.Entry, replaces string) *cache.Entry {
	operator.Replaces = replaces
	return operator
}

func requirePropertiesEqual(t *testing.T, a, b []*api.Property) {
	type Property struct {
		Type  string
		Value interface{}
	}
	nice := func(in *api.Property) Property {
		var i interface{}
		if err := json.Unmarshal([]byte(in.Value), &i); err != nil {
			t.Fatalf("property value %q could not be unmarshaled as json: %s", in.Value, err)
		}
		return Property{
			Type:  in.Type,
			Value: i,
		}
	}
	var l, r []Property
	for _, p := range a {
		l = append(l, nice(p))
	}
	for _, p := range b {
		r = append(r, nice(p))
	}
	require.ElementsMatch(t, l, r)
}

func apiSetToProperties(crds, apis cache.APISet, deprecated bool) (out []*api.Property) {
	out = make([]*api.Property, 0)
	for a := range crds {
		val, err := json.Marshal(opregistry.GVKProperty{
			Group:   a.Group,
			Kind:    a.Kind,
			Version: a.Version,
		})
		if err != nil {
			panic(err)
		}
		out = append(out, &api.Property{
			Type:  opregistry.GVKType,
			Value: string(val),
		})
	}
	for a := range apis {
		val, err := json.Marshal(opregistry.GVKProperty{
			Group:   a.Group,
			Kind:    a.Kind,
			Version: a.Version,
		})
		if err != nil {
			panic(err)
		}
		out = append(out, &api.Property{
			Type:  opregistry.GVKType,
			Value: string(val),
		})
	}
	if deprecated {
		val, err := json.Marshal(opregistry.DeprecatedProperty{})
		if err != nil {
			panic(err)
		}
		out = append(out, &api.Property{
			Type:  opregistry.DeprecatedType,
			Value: string(val),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return
}

func apiSetToDependencies(crds, apis cache.APISet) (out []*api.Dependency) {
	if len(crds)+len(apis) == 0 {
		return nil
	}
	out = make([]*api.Dependency, 0)
	for a := range crds {
		val, err := json.Marshal(opregistry.GVKDependency{
			Group:   a.Group,
			Kind:    a.Kind,
			Version: a.Version,
		})
		if err != nil {
			panic(err)
		}
		out = append(out, &api.Dependency{
			Type:  opregistry.GVKType,
			Value: string(val),
		})
	}
	for a := range apis {
		val, err := json.Marshal(opregistry.GVKDependency{
			Group:   a.Group,
			Kind:    a.Kind,
			Version: a.Version,
		})
		if err != nil {
			panic(err)
		}
		out = append(out, &api.Dependency{
			Type:  opregistry.GVKType,
			Value: string(val),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return
}
