package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "xworld.cn/pkg/client/clientset/versioned"
	"xworld.cn/pkg/client/informers/externalversions"
	"xworld.cn/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

const (
	NUM_WORKER = 2
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	kubeClient, extClient, xworldClient, err := newKubeClient(false)

	// 第二个参数resyncPeriod指每过一段时间，清空本地缓存，从apiserver中做一次list
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	xworldInformerFactory := externalversions.NewSharedInformerFactory(xworldClient, time.Second*30)

	controller := NewController(kubeClient, kubeInformerFactory, extClient, xworldClient, xworldInformerFactory)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	xworldInformerFactory.Start(stopCh)

	if err = controller.Run(NUM_WORKER, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

	<-stopCh
	klog.Info("xworld controller stop")
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func newKubeClient(runInCluster bool) (*kubernetes.Clientset, *extclientset.Clientset, *clientset.Clientset, error) {
	var config *rest.Config
	var err error
	if !runInCluster {
		kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create out-cluster kube cli configuration: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create in-cluster kube cli configuration: %v", err)
		}
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not create the kubernetes clientset: %v", err)
	}
	extClient, err := extclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not create the api extension clientset: %v", err)
	}
	xworldClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not create the xworld api clientset: %v", err)
	}
	return kubeClient, extClient, xworldClient, nil
}

func local_test() {
	client, err := newCustomKubeClient()
	if err != nil {
		fmt.Printf("new kube client error: %v\n", err)
	}
	fmt.Printf("new kube client success\n")

	factory := externalversions.NewSharedInformerFactory(client, 30*time.Second)
	informer := factory.Xworld().V1().XServers()
	lister := informer.Lister()

	stopCh := make(chan struct{})
	factory.Start(stopCh)

	for {
		ret, err := lister.List(labels.Everything())
		if err != nil {
			fmt.Printf("list error: %v\n", err)
		} else {
			for _, scaling := range ret {
				fmt.Println(scaling)
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func newCustomKubeClient() (clientset.Interface, error) {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create out-cluster kube cli configuration: %v", err)
	}

	cli, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom kube client: %v", err)
	}
	return cli, nil
}
