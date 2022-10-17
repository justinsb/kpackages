manifests: manifest-istio manifest-gateway

manifest-istio-next-k8s:
	# How to pick k8s version? --cluster-specific works but relies on a cluster
	# We also have to custom build because it hasn't been released yet (?)
	mkdir bin/
	git clone https://github.com/istio/istio third_party/istio
	cd third_party/istio; go build -o ../../bin/istioctl ./istioctl/cmd/istioctl
	bin/istioctl manifest generate --set profile=minimal --cluster-specific > manifest-istio.yaml

manifest-istio:
	istio-1.15.2/bin/istioctl manifest generate --set profile=minimal > manifest-istio.yaml
	# Deprecations ... is there some way to specify k8s version?
	sed -i -e s@policy/v1beta1@policy/v1@g manifest-istio.yaml
	sed -i -e s@autoscaling/v2beta2@autoscaling/v2@g manifest-istio.yaml
	kubectl create ns istio-system --dry-run=client -oyaml > manifest-istio-ns.yaml
	# TODO: Reduce memory?

manifest-gateway:
	kubectl kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v0.5.0" > manifest-gateway.yaml

install:
	kubectl apply -f manifest-gateway.yaml
	kubectl apply -f manifest-istio-ns.yaml
	kubectl apply -f manifest-istio.yaml
