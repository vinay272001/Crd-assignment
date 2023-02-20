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

	"github.com/vinay272001/Crd-assignment/pkg/apis/phoenix.com/v1alpha1"
	clientset "github.com/vinay272001/Crd-assignment/pkg/client/clientset/versioned"
	informers "github.com/vinay272001/Crd-assignment/pkg/client/informers/externalversions/phoenix.com/v1alpha1"
	listers "github.com/vinay272001/Crd-assignment/pkg/client/listers/phoenix.com/v1alpha1"
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
	klog.Info("NewController is called")
	klog.Info("\n--------------------------------------------------\n")
	controller := &Controller{
		kubeclient:     kubeclient,
		appclient:   appclient,
		applisters:	appInformer.Lister(),
		appSynced:        appInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "App"),
	}
	klog.Info("NewController made")
	klog.Info("\n--------------------------------------------------\n")

	klog.Info("Setting up event handlers")
	klog.Info("\n--------------------------------------------------\n")
	// event handler when the trackPod resources are added/deleted/updated.
	appInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: controller.createHandler,
			UpdateFunc: func(old, new interface{}) {
				klog.Info("In the UpdateFunc method")
				klog.Info("\n--------------------------------------------------\n")
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

	klog.Info("returning controller")
	klog.Info("\n--------------------------------------------------\n")
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
	klog.Info("\n--------------------------------------------------\n")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	klog.Info("\n--------------------------------------------------\n")
	if ok := cache.WaitForCacheSync(ch, c.appSynced); !ok {
		klog.Fatalf("failed to wait for caches to sync")
		klog.Info("\n--------------------------------------------------\n")
	}

	klog.Info("Starting workers")
	klog.Info("\n--------------------------------------------------\n")
	go wait.Until(c.runWorker, time.Second, ch)
	klog.Info("Started workers")
	klog.Info("\n--------------------------------------------------\n")
	<-ch
	klog.Info("Shutting down workers")
	klog.Info("\n--------------------------------------------------\n")

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
		klog.Info("\n--------------------------------------------------\n")
		return false
	}

	// defer c.workqueue.Forget(obj)
	// if err := c.syncHandler(obj.(string)); err != nil {
	// 	klog.Info("\n--------------------------------------------------\n")
	// 	c.workqueue.AddRateLimited(obj.(string))
	// 	// return true
	// }

	// klog.Info("successfully synced ", obj.(string))
	// klog.Info("\n--------------------------------------------------\n")

	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			// c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		// utilruntime.HandleError(err)
		return true
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

	klog.Info("In the syncHandler to sync pods")
	klog.Info("\n--------------------------------------------------\n")

	if err := c.syncPods(app, appList); err != nil {
		klog.Fatalf("Error while syncing the current vs desired state for App %v: %v\n", app.Name, err.Error())
		klog.Info("\n--------------------------------------------------\n")
	}
	
	klog.Info("synced pods successfully")
	klog.Info("\n--------------------------------------------------\n")
	

	err = c.waitForPods(app, appList)
	if err != nil {
		klog.Fatalf("error %s, waiting for pods to meet the expected state", err.Error())
		klog.Info("\n--------------------------------------------------\n")
	}

	klog.Info("successfully waited for pods")
	klog.Info("\n--------------------------------------------------\n")

	err = c.updateAppStatus(app, app.Spec.Message, appList)
	if err != nil {
		return err
	}

	klog.Info("successfully updated status")
	klog.Info("\n--------------------------------------------------\n")

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
		klog.Info("pods are getting deleted ", numDelete)
		klog.Info("\n--------------------------------------------------\n")
		for i:= 0; i < numDelete ; i++ {
			err := c.kubeclient.CoreV1().Pods(app.Namespace).Delete(context.TODO(), appList.Items[i].Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Fatalf("error while deleting the pods %v", app.Name)
				klog.Info("\n--------------------------------------------------\n")
				
				return err
			}
		}
		klog.Info("pods deleted successfully")
		klog.Info("\n--------------------------------------------------\n")
	}

	if ifCreate {
		klog.Info("pods are getting created ", numCreate)
		klog.Info("\n--------------------------------------------------\n")
		for i := 0; i < numCreate ; i++ {
			newApp, err := c.kubeclient.CoreV1().Pods(app.Namespace).Create(context.TODO(), newPod(app), metav1.CreateOptions{})
			if err != nil {
				if errors.IsAlreadyExists(err) {
					numCreate++
				} else {
					klog.Fatalf("error in creating pods %v", app.Name)
					klog.Info("\n--------------------------------------------------\n")
					return err
				}
			} 
			if newApp.Name != "" {
				klog.Info("Created!")
				klog.Info("\n--------------------------------------------------\n")
			}
		}
	}

	return nil
}

func (c *Controller) waitForPods(app *v1alpha1.App, appsList *corev1.PodList) error {
	// klog.Info("waiting for pods to be in running state")
	// klog.Info("\n--------------------------------------------------\n")
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
	klog.Info("calculating total number of running pods")
	klog.Info("\n--------------------------------------------------\n")
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

	klog.Info("Total number of running pods are ", currentPods)

	return currentPods
}

func (c *Controller) updateAppStatus(app *v1alpha1.App, message string, appsList *corev1.PodList) error {

	currentPods := c.getCurrentPods(app)
	appCopy := app.DeepCopy()

	appCopy.Status.Count = currentPods
	appCopy.Status.Message = message

	klog.Info("updating status")
	_, err := c.appclient.PhoenixV1alpha1().Apps(app.Namespace).UpdateStatus(context.TODO(), appCopy, metav1.UpdateOptions{})

	return err
}

func newPod(app *v1alpha1.App) *corev1.Pod {
	klog.Info("new pods creation function")
	klog.Info("\n--------------------------------------------------\n")
	labels := map[string]string{
		"controller": app.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name: fmt.Sprintf(app.Name + "-" + strconv.Itoa(rand.Intn(10000))),
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
						{
							Name: "COUNT",
							Value: strconv.Itoa(int(*app.Spec.Count)),
						},
					},
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						"echo 'Message = $(MESSAGE) and Count = $(COUNT)'",
					},
				},

			},
		},
	}
} 


func (c *Controller) createHandler(obj interface{}) {
	klog.Info("In the createHandler")
	klog.Info("\n--------------------------------------------------\n")
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func(c *Controller) deleteHandler(obj interface{}) {
	klog.Info("Deleting pods using kubeClient")
	klog.Info("\n--------------------------------------------------\n")
	app, ok := obj.(*v1alpha1.App)
    if !ok {
        return
    }

	klog.Info("got app")
	klog.Info("\n--------------------------------------------------\n")
    // Delete the Pods associated with the custom resource

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": app.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

    podList, err := c.kubeclient.CoreV1().Pods(app.Namespace).List(context.TODO(), listOptions)

	klog.Info("got pods list")
	klog.Info("\n--------------------------------------------------\n")

    if err != nil {
        klog.Errorf("Failed to list pods for app %s: %v", app.Name, err)
        return
    }
	klog.Info("deleting pods")
	klog.Info("\n--------------------------------------------------\n")
    for _, pod := range podList.Items {
        if err := c.kubeclient.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
            klog.Errorf("Failed to delete pod %s: %v", pod.Name, err)
        }
    }
	klog.Info("pods deleted")
	klog.Info("\n--------------------------------------------------\n")
}