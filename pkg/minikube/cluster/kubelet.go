package cluster

import (
	"bytes"
	"fmt"
	"text/template"

	"k8s.io/minikube/pkg/minikube/constants"
)

// StartKubeletCmd returns the command line invocation for initializing
// and starting a Kubelet systemd unit.
func StartKubeletCmd(kCfg KubernetesConfig) (string, error) {
	kubeletSvc, err := renderKubeletService(kCfg)
	if err != nil {
		return "", fmt.Errorf("failed to start kubelet: %v", err)
	}

	injectSvcCmd := fmt.Sprintf("printf %%s \"%s\" | sudo tee %s\n", kubeletSvc, constants.KubeletServicePath)
	startSvcCmd := `sudo systemctl daemon-reload
			sudo systemctl enable kubelet.service
			sudo systemctl restart kubelet.service || true`
	return injectSvcCmd + startSvcCmd, nil
}

func renderKubeletService(kCfg KubernetesConfig) (string, error) {
	t, err := template.New("kubeletService").Parse(kubeletSystemdTmpl)
	if err != nil {
		return "", fmt.Errorf("bad kubelet base configuration: %v", err)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, &kCfg); err != nil {
		return "", fmt.Errorf("could not configure kubelet systemd unit: %v", err)
	}
	return out.String(), nil
}

var kubeletSystemdTmpl = `[Unit]
Description=Kubelet via Hyperkube ACI

[Service]
Environment="RKT_RUN_ARGS=--uuid-file-save=/var/run/kubelet-pod.uuid \
  --volume=resolv,kind=host,source=/etc/resolv.conf \
  --mount volume=resolv,target=/etc/resolv.conf \
  --volume var-lib-cni,kind=host,source=/var/lib/cni \
  --mount volume=var-lib-cni,target=/var/lib/cni \
  --volume var-log,kind=host,source=/var/log \
  --mount volume=var-log,target=/var/log \
  --volume etc-kubernetes,kind=host,source=/etc/kubernetes,readOnly=false \
  --volume etc-ssl-certs,kind=host,source=/etc/ssl/certs,readOnly=true \
  --volume usr-share-certs,kind=host,source=/usr/share/ca-certificates,readOnly=true \
  --volume var-lib-docker,kind=host,source=/var/lib/docker,readOnly=false \
  --volume var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=false,recursive=true \
  --volume os-release,kind=host,source=/usr/lib/os-release,readOnly=true \
  --volume run,kind=host,source=/run,readOnly=false \
  --mount volume=etc-kubernetes,target=/etc/kubernetes \
  --mount volume=etc-ssl-certs,target=/etc/ssl/certs \
  --mount volume=usr-share-certs,target=/usr/share/ca-certificates \
  --mount volume=var-lib-docker,target=/var/lib/docker \
  --mount volume=var-lib-kubelet,target=/var/lib/kubelet \
  --mount volume=os-release,target=/etc/os-release \
  --mount volume=run,target=/run"

ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests \
  /srv/kubernetes/manifests /etc/kubernetes/checkpoint-secrets \
  /etc/kubernetes/cni/net.d /var/lib/cni \
  /var/lib/docker /var/lib/kubelet /run/kubelet
ExecStartPre=-/bin/rkt rm --uuid-file=/var/run/kubelet-pod.uuid

ExecStart=/bin/rkt run ${RKT_RUN_ARGS} \
  docker://gcr.io/google_containers/hyperkube-amd64:{{.KubernetesVersion}} \
  --exec=/kubelet \
  -- \
  --tls-cert-file=/var/lib/localkube/certs/apiserver.crt \
  --tls-private-key-file=/var/lib/localkube/certs/apiserver.key \
  --cni-conf-dir=/etc/kubernetes/cni/net.d \
  --network-plugin=cni \
  --lock-file=/var/run/lock/kubelet.lock \
  --exit-on-lock-contention \
  --pod-manifest-path=/etc/kubernetes/manifests \
  --allow-privileged \
  --minimum-container-ttl-duration=6m0s \
  --cluster-domain={{.DNSDomain}} \
  --client-ca-file=/var/lib/localkube/certs/ca.crt \
  --anonymous-auth=false
ExecStop=-/bin/rkt stop --uuid-file=/var/run/kubelet-pod.uuid

Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target`
