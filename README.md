# Crd-assignment

## Resource Creation 

- For starting minikube `$ minikube start`
- Resource `Type = App, Group = phoenix.com, version = v1aplha1`
- Create resource file `app.yaml` with Spec details
- Added resource definition by `$ kubectl create -f artifacts/apps/crd.yaml`
- Creating an object of CRD by declaring `app.yaml`
- Added crd based object by `$ kubectl create -f artifacts/apps/app.yaml`
- Checking if object is running by `$ kubeclt get app`
```
NAME            AGE
phoenix-app-1   43s
```

## Controller Creation

- point to the directory: `cd home/{user}/go/src/github.com/{github-username}/Crd-assignment
- create folder `pkg/apis/{Group}/{version}` and add types.go file
- Add definitions of cluster type and its fields(Spec, Status, List) in the `types.go` file
- We need to specifiy that the Kluster Spec is a Kubernetes object, for this we will create a `register.go` file in the same directory that will register the Kluster type to scheme.
- Create `doc.go` for declaring tabs. `tabs` are basically used to call a particular instruction for all valid instance over the codebase. For eg- `+k8s:deepcopy-gen=package` means that a deep copy must be generated at every package. This is global. If we declare it anywhere else, it will be local. 
- Create register.go in the directory `pkg/apis/{Group}` as every directory must contain a go file.
- Install `code-generator` by `$ go get k8s.io/code-generator` 
- Create alias path `$ execDir=~/go/pkg/mod/k8s.io/code-generator@v0.26.1`
- Create deepcopy, lister, clientset and informers by runnning `$ "${execDir}"/generate-groups.sh all github.com/vinay272001/Crd-assignment/pkg/client github.com/vinay272001/Crd-assignment/pkg/apis phoenix.com:v1alpha1 --go-header-file "${execDir}"/hack/boilerplate.go.txt`
- Create the `main.go`. In `main.go` we will declare the `kubeconfig`, create `clientsets` and the generate clusters.
- To add all dependency : `$ go mod tidy`
- Create the `controller.go`.

## Running Controller

- build the controller using `$ go build -o Crd-assignment` 
- To start the controller `$ ./crd-assignment
- When custom resource is created using `kubectl create -f app.yaml` number of running pods will sync with the count field in CR and the pods will echo message and count in it
- For deleting a CR
- `$ kubectl delete -f artifacts/apps/app.yml`, this will make sure all the associated pods are deleted as well.

For deleting a pod
- `$ kubectl delete pod {podname}`
After deleting the pod, they won't spawn back

To get podname
- `$ kubectl get pods`