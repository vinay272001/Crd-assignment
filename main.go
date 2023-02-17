package main

import (
	"flag"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"

	clientset "github.com/vinay272001/Crd-assignment/pkg/client/clientset/versioned"
	informers "github.com/vinay272001/Crd-assignment/pkg/client/informers/externalversions"
)

func main() {
	klog.InitFlags(nil)
	var kubeconfig *string

	klog.Info("Searching kubeConfig")
	klog.Info("\n--------------------------------------------------\n")

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), 
		"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	klog.Info("Building config from the kubeConfig")
	klog.Info("\n--------------------------------------------------\n")

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		klog.Fatalf("Building config from flags failed, %s, trying to build inclusterconfig", err.Error())
		cfg, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalf("error %s building inclusterconfig", err.Error())
		}
	}

	klog.Info("getting the appClient")
	klog.Info("\n--------------------------------------------------\n")

	appClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	klog.Info("getting k8s client")
	klog.Info("\n--------------------------------------------------\n")

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	appInformerFactory := informers.NewSharedInformerFactory(appClient, time.Second*30)

	ch := make(chan struct{})
	controllerObj := NewController(kubeClient, appClient, 
		appInformerFactory.Phoenix().V1alpha1().Apps())

	klog.Info("got controller in main.go")
	klog.Info("\n--------------------------------------------------\n")

	klog.Info("starting channel and controller.run()")
	klog.Info("\n--------------------------------------------------\n")
	
	appInformerFactory.Start(ch)
	if err := controllerObj.Run(ch); err != nil {
		klog.Fatalf("error running controller %s\n", err)
	}
}