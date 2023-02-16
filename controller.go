package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/vinay272001/Crd-assignment/pkg/apis/phoenix.io/v1alpha1"
	clientset "github.com/vinay272001/Crd-assignment/pkg/client/clientset/versioned"
	informers "github.com/vinay272001/Crd-assignment/pkg/client/informers/externalversions/phoenix.io/v1alpha1"
	listers "github.com/vinay272001/Crd-assignment/pkg/client/listers/phoenix.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"github.com/kanisterio/kanister/pkg/poll"
)

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by ExampleCrd"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "ExampleCr synced successfully"
)

type Controller struct {
	// kubeclient is a standard kubernetes clientset
	kubeclient kubernetes.Interface
	// appclient is a clientset for our own API group
	appclient clientset.Interface

	applisters        listers.AppLister
	appSynced        cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
}

// returns a new controller
func NewController(
	kubeclient kubernetes.Interface, 
	appclient clientset.Interface, 
	appInformer informers.AppInformer) *Controller {
	
	controller := &Controller{
		kubeclient:     kubeclient,
		appclient:   appclient,
		applisters:	appInformer.Lister(),
		appSynced:        appInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "App"),
	}

	klog.Info("Setting up event handlers")
	// event handler when the trackPod resources are added/deleted/updated.
	appInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: controller.createHandler,
			UpdateFunc: func(old, new interface{}) {
				oldApp := old.(*v1alpha1.App)
				newApp := new.(*v1alpha1.App)
				if oldApp == newApp {
					return
				}
				controller.createHandler(new)
			},
			DeleteFunc: controller.deleteHandler,
		},
	)

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ch chan struct{}) error {
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting App controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ch, c.appSynced); !ok {
		klog.Fatalf("failed to wait for caches to sync")
	}

	// klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	// for i := 0; i < workers; i++ {
	// 	go wait.Until(c.runWorker, time.Second, stopCh)
	// }
	go wait.Until(c.runWorker, time.Second, ch)
	klog.Info("Started workers")
	<-ch
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		klog.Info("Shutting down")
		return false
	}

	defer c.workqueue.Forget(obj)
	if err := c.syncHandler(obj.(string)); err != nil {
		klog.Fatalf("Error while syncing the current vs desired state")
		return false
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	app, err := c.applisters.Apps(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": app.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	appList, err := c.kubeclient.CoreV1().Pods(app.Namespace).List(context.TODO(), listOptions)

	if err := c.syncPods(app, appList); err != nil {
		klog.Fatalf("Error while syncing the current vs desired state for App %v: %v\n", app.Name, err.Error())
	}
	

	err = c.waitForPods(app, appList)
	if err != nil {
		klog.Fatalf("error %s, waiting for pods to meet the expected state", err.Error())
	}
	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err = c.updateAppStatus(app, app.Spec.Message, appList)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) syncPods(app *v1alpha1.App, appList *corev1.PodList) error {
	newPods := app.Spec.Count
	currentPods := c.getCurrentPods(app)
	newMessage := app.Spec.Message
	currentMessage := app.Status.Message
	var ifDelete, ifCreate bool
	numCreate := int(*newPods)
	numDelete := 0

	if int(*newPods) != currentPods || newMessage != currentMessage {
		if newMessage != currentMessage {
			ifDelete = true
			ifCreate = true
			numCreate = int(*newPods)
			numDelete = currentPods
		} else {
			if currentPods < int(*newPods) {
				ifCreate = true
				numCreate = int(*newPods) - currentPods
			} else if currentPods > int(*newPods) {
				ifDelete = true
				numDelete = currentPods - int(*newPods)
			}
		}
	}

	if ifDelete {
		for i:= 0; i < numDelete ; i++ {
			err := c.kubeclient.CoreV1().Pods(app.Namespace).Delete(context.TODO(), appList.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Fatalf("")
				return err
			}
		}
	}

	if ifCreate {
		for i := 0; i < numCreate ; i++ {
			newApp, err := c.kubeclient.CoreV1().Pods(app.Namespace).Create(context.TODO(), newPod(app), metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					numCreate++
				} else {
					klog.Fatalf("error in ifcreate")
					return err
				}
			} 
			if newApp.Name != "" {
				klog.Fatalf("Created!")
			}
		}
	}

	return nil
}

func (c *Controller) waitForPods(app *v1alpha1.App, appsList *corev1.PodList) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		currentPods := c.getCurrentPods(app)

		if currentPods == int(*app.Spec.Count) {
			return true, nil
		}
		return false, nil
	})
}

func (c *Controller) getCurrentPods(app *v1alpha1.App) int {
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": app.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	appsList, _ := c.kubeclient.CoreV1().Pods(app.Namespace).List(context.TODO(), listOptions)
	
	currentPods := 0

	for _,pod := range appsList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.Phase == "Running" {
			currentPods++
		}
	}

	return currentPods
}

func (c *Controller) updateAppStatus(app *v1alpha1.App, message string, appsList *corev1.PodList) error {

	appCopy, err := c.appclient.PhoenixV1alpha1().Apps(app.Namespace).Get(context.TODO(), app.Name, metav1.GetOptions{})
	currentPods := c.getCurrentPods(app)
	if err != nil {
		return err
	}

	appCopy.Status.Count = currentPods
	appCopy.Status.Message = message

	_, err = c.appclient.PhoenixV1alpha1().Apps(app.Namespace).UpdateStatus(context.Background(), appCopy, metav1.UpdateOptions{})

	return err


}

func newPod(app *v1alpha1.App) *corev1.Pod {
	labels := map[string]string{
		"controller": app.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name: fmt.Sprintf(app.Name + "-" + strconv.Itoa(rand.Intn(100000000))),
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, v1.SchemeGroupVersion.WithKind("App")),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "nginx",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name: "MESSAGE",
							Value: app.Spec.Message,
						},
					},
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						"while true; do echo '$(MESSAGE)'; sleep 100; done",
					},
				},

			},
		},
	}
} 


func (c *Controller) createHandler(obj interface{}) {
	klog.Fatalf("create Handler here")
	c.workqueue.Add(obj)
}

func (c *Controller) deleteHandler(obj interface{}) {
	klog.Fatalf("")
	c.workqueue.Done(obj)
}