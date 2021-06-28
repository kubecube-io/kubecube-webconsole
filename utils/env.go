package utils

import "os"

func GetKubeCubeSvc() string {
	svc := os.Getenv("KUBECUBE_SVC")
	if svc != "" {
		return svc
	}
	return "kubecube-nodeport.kubecube-system:7443"
}

func getJwtSecret() string {
	return os.Getenv("JWT_SECRET")
}
