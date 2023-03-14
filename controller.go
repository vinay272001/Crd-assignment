package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/vinay272001/Crd-assignment/kmeta"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/kanisterio/kanister/pkg/poll"
	"github.com/vinay272001/Crd-assignment/pkg/apis/phoenix.com/v1alpha1"
	clientset "github.com/vinay272001/Crd-assignment/pkg/client/clientset/versioned"
	informers "github.com/vinay272001/Crd-assignment/pkg/client/informers/externalversions/phoenix.com/v1alpha1"
	listers "github.com/vinay272001/Crd-assignment/pkg/client/listers/phoenix.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
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
	// pipelineRunClient is a clientset for our own API group
	pipelineRunClient clientset.Interface

	pipelineRunListers        listers.PipelineRunLister
	pipelineRunSynced        cache.InformerSynced

	taskRunListers        listers.TaskRunLister

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
	pipelineRunClient clientset.Interface, 
	pipelineRunInformer informers.PipelineRunInformer,
	taskRunInformer informers.TaskRunInformer) *Controller {
	klog.Info("NewController object creation Here")
	klog.Info("\n--------------------------------------------------\n")
	controller := &Controller{
		kubeclient:     kubeclient,
		pipelineRunClient:   pipelineRunClient,
		pipelineRunListers:	pipelineRunInformer.Lister(),
		pipelineRunSynced:        pipelineRunInformer.Informer().HasSynced,
		taskRunListers:	taskRunInformer.Lister(),
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PipelineRun"),
	}
	klog.Info("NewController created")
	klog.Info("\n--------------------------------------------------\n")

	klog.Info("Setting up event handlers")
	klog.Info("\n--------------------------------------------------\n")
	pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: controller.createHandler,
			UpdateFunc: func(old, new interface{}) {
				klog.Info("UpdateFunc method Here")
				klog.Info("\n--------------------------------------------------\n")
				oldApp := old.(*v1alpha1.PipelineRun)
				newApp := new.(*v1alpha1.PipelineRun)
				if oldApp == newApp {
					return
				}
				controller.createHandler(new)
			},
			DeleteFunc: controller.deleteHandler,
		},
	)

	taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: kmeta.FilterController(&v1alpha1.PipelineRun{}),
	})
	taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{

		DeleteFunc: controller.taskDeleteHandler,

	})
	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ch chan struct{}) error {
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting pipelineRun controller")
	klog.Info("\n--------------------------------------------------\n")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	klog.Info("\n--------------------------------------------------\n")
	if ok := cache.WaitForCacheSync(ch, c.pipelineRunSynced); !ok {
		klog.Fatalf("failed to wait for caches to sync")
		klog.Info("\n--------------------------------------------------\n")
	}

	klog.Info("Starting workers")
	klog.Info("\n--------------------------------------------------\n")
	go wait.Until(c.runWorker, time.Second, ch)
	klog.Info("workers started")
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
		klog.Fatalf("Shutting down")
		return false
	}

	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		// defer c.workqueue.Done(obj)
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		key, ok := obj.(string)
		if !ok {
			klog.Fatalf("error while calling Namespace Key func on cache")
			
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
		klog.Info("\n--------------------------------------------------\n")
		return nil
	}(obj)

	if err != nil {
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the pipelineRun resource with this namespace/name
	pipelineRun, err := c.pipelineRunListers.PipelineRuns(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	
	klog.Info("In the syncHandler to create taskRun")
	klog.Info("\n--------------------------------------------------\n")

	klog.Info("checking if newTask is needed or not")
	klog.Info("\n--------------------------------------------------\n")

	var taskRun *v1alpha1.TaskRun

	if pipelineRun.Spec.Message != pipelineRun.Status.Message || pipelineRun.Spec.Count != pipelineRun.Status.Count {
		
		taskRun, err = c.syncPipelineTask(pipelineRun)

	} else{
		klog.Info("pipelineRun not updated")
		klog.Info("\n--------------------------------------------------\n")
	}

	if err != nil {
		klog.Fatalf("error creating new taskRun for current pipelineRun", err.Error())
	}
	
	if taskRun != nil {
		
		c.updateStatus(pipelineRun, taskRun)

	}

	return nil
}


func(c *Controller) updateStatus (pipelineRun *v1alpha1.PipelineRun, taskRun *v1alpha1.TaskRun) {

	klog.Info("taskRun created successfully")
	klog.Info("\n--------------------------------------------------\n")

	if err := c.waitForPods(taskRun); err != nil {
		klog.Errorf("error %s, waiting for pods to meet the expected state", err.Error())
	}
	// update taskrun status
	if err := c.updateTaskRunStatus(taskRun); err != nil {
		klog.Errorf("error %s updating PipelineRun status", err.Error())
	}

	// update pipelinerun status

	klog.Info(pipelineRun.Status)

	if err := c.updatePipelineRunStatus(pipelineRun, taskRun); err != nil {
		klog.Errorf("error %s updating PipelineRun status", err.Error())
	}
	
	klog.Info(pipelineRun.Status)

	if pipelineRun.Status.Count == c.getCompletedPods(taskRun) {
		var obj interface{} = taskRun
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			klog.Fatalf("error while calling Namespace Key func on cache", err.Error())
		
			return
		}
		c.workqueue.Done(key)
	}

}


func (c *Controller) syncPipelineTask(pipelineRun *v1alpha1.PipelineRun) (*v1alpha1.TaskRun, error) {

	klog.Info("New task Needed!")
	klog.Info("\n--------------------------------------------------\n")
	taskRun, err := c.pipelineRunClient.PhoenixV1alpha1().TaskRuns(pipelineRun.Namespace).Create(context.TODO(), createTaskRun(pipelineRun), metav1.CreateOptions{})
	if err != nil {
		klog.Fatalf("error creating new taskRun", err.Error())
		return nil, err
	}
	if taskRun != nil {
		klog.Info("new taskRun has been created")
		klog.Info("\n--------------------------------------------------\n")
		err := c.executeTaskRun(pipelineRun, taskRun)
		if err != nil {
			klog.Fatalf("error executing taskRun", err.Error())
		}
	}
	return taskRun, nil
}

func createTaskRun(pipelineRun *v1alpha1.PipelineRun) *v1alpha1.TaskRun {
	return &v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-taskrun-", pipelineRun.Name),
			// Name: fmt.Sprintf("%v-taskrun-%v", pipelineRun.Name, pipelineRun.ObjectMeta.Generation),
			Namespace: pipelineRun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pipelineRun, v1alpha1.SchemeGroupVersion.WithKind("PipelineRun")),
			},
		},
		Spec: v1alpha1.TaskRunSpec{
			Message: pipelineRun.Spec.Message,
			Count: pipelineRun.Spec.Count,
		},
	}
}

func (c *Controller) executeTaskRun(pipelineRun *v1alpha1.PipelineRun, taskRun *v1alpha1.TaskRun) (error){
	toCreate := taskRun.Spec.Count

	for i := 0; i < toCreate; i++ {
		newPod, err := c.kubeclient.CoreV1().Pods(taskRun.Namespace).Create(context.TODO(), newPod(taskRun), metav1.CreateOptions{})
		if err != nil {
			klog.Fatalf("error creating pod for taskRun ", taskRun.Name)
		}
		if newPod.Name != "" {
			klog.Info("Pod created successfully ", newPod.Name)
			klog.Info("\n--------------------------------------------------\n")
		}

	}
	return nil
}


func (c *Controller) waitForPods(taskRun *v1alpha1.TaskRun) error {
	klog.Info("waiting for pods to be in complete state")
	klog.Info("\n--------------------------------------------------\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		currentPods := c.getCompletedPods(taskRun)

		if currentPods == taskRun.Spec.Count {
			return true, nil
		}
		return false, nil
	})
}

func (c *Controller) getCompletedPods(taskRun *v1alpha1.TaskRun) int {
	// klog.Info("calculating total number of running pods")
	// klog.Info("\n--------------------------------------------------\n")
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"controller": taskRun.Name,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podList, _ := c.kubeclient.CoreV1().Pods(taskRun.Namespace).List(context.TODO(), listOptions)
	
	completedPods := 0

	for _,pod := range podList.Items {
		if pod.ObjectMeta.DeletionTimestamp.IsZero() && pod.Status.Phase == "Succeeded" {
			completedPods++
		}
	}

	klog.Info("completed pods: ", completedPods)
	klog.Info("\n--------------------------------------------------\n")

	return completedPods
}

func (c *Controller) updatePipelineRunStatus(pipelineRun *v1alpha1.PipelineRun, taskRun *v1alpha1.TaskRun) error {

	pipelineRunCopy, err := c.pipelineRunClient.PhoenixV1alpha1().PipelineRuns(pipelineRun.Namespace).Get(context.Background(), pipelineRun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	taskRunCopy, err := c.pipelineRunClient.PhoenixV1alpha1().TaskRuns(taskRun.Namespace).Get(context.Background(), taskRun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	pipelineRunCopy.Status.Message = taskRunCopy.Status.Message
	pipelineRunCopy.Status.Count = taskRunCopy.Status.Count

	klog.Info("updating status")
	klog.Info("\n--------------------------------------------------\n")
	_, err = c.pipelineRunClient.PhoenixV1alpha1().PipelineRuns(pipelineRun.Namespace).UpdateStatus(context.TODO(), pipelineRunCopy, metav1.UpdateOptions{})

	return err
}

func (c *Controller) updateTaskRunStatus(taskRun *v1alpha1.TaskRun) error {

	currentPods := c.getCompletedPods(taskRun)
	taskRunCopy, err := c.pipelineRunClient.PhoenixV1alpha1().TaskRuns(taskRun.Namespace).Get(context.Background(), taskRun.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	taskRunCopy.Status.Count = currentPods
	taskRunCopy.Status.Message = taskRunCopy.Spec.Message

	klog.Info("updating status")
	klog.Info("\n--------------------------------------------------\n")
	_, err = c.pipelineRunClient.PhoenixV1alpha1().TaskRuns(taskRun.Namespace).UpdateStatus(context.TODO(), taskRunCopy, metav1.UpdateOptions{})

	return err
}

func newPod(taskRun *v1alpha1.TaskRun) *corev1.Pod {
	klog.Info("new pods creation function")
	klog.Info("\n--------------------------------------------------\n")
	labels := map[string]string{
		"controller": taskRun.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			GenerateName: fmt.Sprintf("%s-", taskRun.Name),
			Namespace: taskRun.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(taskRun, v1alpha1.SchemeGroupVersion.WithKind("TaskRun")),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: "Never",
			Containers: []corev1.Container{
				{
					Name: "nginx",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{
							Name: "MESSAGE",
							Value: taskRun.Spec.Message,
						},
						{
							Name: "COUNT",
							Value: strconv.Itoa(taskRun.Spec.Count),
						},
					},
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						"echo 'Message is $(MESSAGE) where Count is $(COUNT)'",
					},
				},

			},
		},
	}
} 


func (c *Controller) createHandler(obj interface{}) {
	klog.Info("In the createHandler")
	klog.Info("\n--------------------------------------------------\n")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Fatalf("error while calling Namespace Key func on cache", err.Error())
		
		return
	}
	c.workqueue.Add(key)
}

func(c *Controller) deleteHandler(obj interface{}) {
	klog.Info("Deleting pods")
	klog.Info("\n--------------------------------------------------\n")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Fatalf("error while calling Namespace Key func on cache", err.Error())
		
		return
	}
	c.workqueue.Done(key)
}

func(c *Controller) getOwnerPipelineRun(taskRun *v1alpha1.TaskRun) (*v1alpha1.PipelineRun, error) {
    pipelineRunRef := metav1.GetControllerOf(taskRun)
    if pipelineRunRef == nil {
        return nil, fmt.Errorf("TaskRun %s/%s has no owner PipelineRun", taskRun.Namespace, taskRun.Name)
    }

    pipelineRun, err := c.pipelineRunClient.PhoenixV1alpha1().PipelineRuns(taskRun.Namespace).Get(context.Background(), pipelineRunRef.Name, metav1.GetOptions{})
    if err != nil {
        return nil, fmt.Errorf("Error retrieving owner PipelineRun %s/%s: %s", taskRun.Namespace, pipelineRunRef.Name, err.Error())
    }

    return pipelineRun, nil
}


func (c *Controller) taskDeleteHandler(obj interface{}) {
	klog.Info("In the deleteHandlerTest")
	klog.Info("\n--------------------------------------------------\n")

	taskRun, ok := obj.(*v1alpha1.TaskRun)
    if !ok {
        return
    }

	// const resyncPeriod = 30 * time.Second

	// Ignore deleted objects with a deletion timestamp older than our resync period
    // if !taskRun.DeletionTimestamp.IsZero() && taskRun.DeletionTimestamp.Time.Before(time.Now().Add(-resyncPeriod)) {
    //     return
    // }

	pipelineRun, err := c.getOwnerPipelineRun(taskRun)

	if err != nil {
		klog.Info("PipelineRun Deleted")
		klog.Info("\n--------------------------------------------------\n")
	} else {
		var obj interface{} = pipelineRun
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			klog.Fatalf("error while calling Namespace Key func on cache", err.Error())
		
			return
		}
		c.workqueue.Add(key)
	}

}