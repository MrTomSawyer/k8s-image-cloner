# Image Clone Controller (ICC)

The Image Clone Controller (ICC) is a Kubernetes controller that monitors Deployments and DaemonSets, backing up Docker container images to a backup registry.

## Prerequisites

- A Kubernetes cluster (Minikube can be used for local testing);
- Docker installed on your local machine;
- Docker Hub account for pushing container images;

## Getting Started

### 1) Building and Pushing the Docker Image

First, build the Docker image for the controller and push it to your Docker Hub account.

```bash
# Build the Docker image for the ICC controller
docker build -t your_dockerhub_username/image-cloner .

# Push the image to your Docker Hub repository
docker push your_dockerhub_username/image-cloner:tag
```
### 2) Configuring the Kubernetes Cluster

- Apply RBAC Configuration.

To ensure that the controller has the necessary permissions, create a ServiceAccount, Role, and RoleBinding by applying the following RBAC configuration

```bash
kubectl apply -f ./k8s/rbac
```
- Create Kubernetes Secret for Docker Credentials

The ICC requires access to your Docker registry to back up container images. You need to create a Kubernetes secret with your Docker Hub credentials. Update the ./k8s/secrets/docker-cred-secret.yml file with your Docker Hub username and password encoded in Base64.

Once updated, apply the secret:

```bash
kubectl apply -f ./k8s/secrets
```

- Deploy the ICC Controller

Now, deploy the ICC controller to your Kubernetes cluster:

```bash
kubectl apply -f ./k8s/setup/cloner-deployment.yml
```

### 3) Deploying Example Applications

- Deploy example applications (a Deployment and a DaemonSet) to test the ICC:

```bash
# Deploy a test application (Deployment)
kubectl apply -f ./k8s/setup/app-deployment.yml

# Deploy a test application (DaemonSet)
kubectl apply -f ./k8s/setup/app-daemonset.yml
```

### 4) Verifying the Controller

To verify that the ICC is working correctly, you can view the logs of the ICC controller:

```bash
kubectl logs deployment.apps/image-cloner-deployment
```

If the image is successfully cloned, the logs will show output similar to:

```bash
2024-10-07T14:03:44Z	INFO	DaemonSet reconciler	starting reconciliation	                {"namespace": "default", "image": "busybox"}
2024-10-07T14:03:44Z	INFO	Image Cloner	        starting cloning process:	            {"image": "busybox"}
2024-10-07T14:03:58Z	INFO	DaemonSet reconciler	reconciliation successfully finished	{"namespace": "default"}
```

## Notes
- The controller ignores any resources in the kube-system namespace;
- The controller does not clone images that have already been cloned (those have a prefix with a backup registry name);
- There are finalizers included in the manifests for deployments and daemonsets to ensure that these entities cannot be deleted until their container images have been successfully copied;