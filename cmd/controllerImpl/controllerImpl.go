package controllerImpl

import (
	controller "github.com/anandnilkal/aws-services/pkg/controller"
	"time"

	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientset "github.com/anandnilkal/aws-services/pkg/generated/clientset/versioned"
	informers "github.com/anandnilkal/aws-services/pkg/generated/informers/externalversions"
	"github.com/anandnilkal/aws-services/pkg/signals"
	"github.com/anandnilkal/aws-services/cmd/resource"
)

func CreateController(cfg *restclient.Config) {

	resourceHandler := resource.NewStreamHandler()
	stopCh := signals.SetupSignalHandler()
	awsServicesStreamClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building awsServicesStream clientset: %s", err.Error())
	}

	awsServicesStreamInformerFactory := informers.NewSharedInformerFactory(awsServicesStreamClient, time.Second*30)
	ctrl := controller.NewControllerFactory("Stream", resourceHandler.AddFunc, resourceHandler.DeleteFunc, resourceHandler.UpdateFunc, awsServicesStreamInformerFactory.Awsservices().V1alpha1().Streams(), awsServicesStreamClient)
	awsServicesStreamInformerFactory.Start(stopCh)
	if err := ctrl.Run(1, stopCh); err != nil {
		klog.Fatalf("failue to run controller: %s", err.Error())
	}
}
