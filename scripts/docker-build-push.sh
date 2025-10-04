#!/bin/bash

set -e

# Configuration
DOCKER_REGISTRY=${DOCKER_USERNAME:-moabdelazem}
TAG=${1:-latest}

echo -e "${YELLOW}Building and pushing images to ${DOCKER_REGISTRY}${NC}"
echo -e "${YELLOW}Tag: ${TAG}${NC}\n"

# Check Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    exit 1
fi

# Build and push auth service
echo -e "\n${GREEN}[1/3] Auth Service${NC}"
docker build -t ${DOCKER_REGISTRY}/microservices-auth:${TAG} -t ${DOCKER_REGISTRY}/microservices-auth:latest -f ./auth/Dockerfile ./auth
docker push ${DOCKER_REGISTRY}/microservices-auth:${TAG}
docker push ${DOCKER_REGISTRY}/microservices-auth:latest

# Build and push tasks service
echo -e "\n${GREEN}[2/3] Tasks Service${NC}"
docker build -t ${DOCKER_REGISTRY}/microservices-tasks:${TAG} -t ${DOCKER_REGISTRY}/microservices-tasks:latest -f ./tasks/Dockerfile ./tasks
docker push ${DOCKER_REGISTRY}/microservices-tasks:${TAG}
docker push ${DOCKER_REGISTRY}/microservices-tasks:latest

# Build and push client
echo -e "\n${GREEN}[3/3] Client${NC}"
docker build \
    --build-arg VITE_AUTH_SERVICE_URL=http://auth-service/api/auth \
    --build-arg VITE_TASKS_SERVICE_URL=http://tasks-service/api/tasks \
    -t ${DOCKER_REGISTRY}/microservices-client:${TAG} \
    -t ${DOCKER_REGISTRY}/microservices-client:latest \
    -f ./client/Dockerfile ./client
docker push ${DOCKER_REGISTRY}/microservices-client:${TAG}
docker push ${DOCKER_REGISTRY}/microservices-client:latest

# Done
echo -e "\n${GREEN}Done! Pushed:${NC}"
echo "  ${DOCKER_REGISTRY}/microservices-auth:${TAG}"
echo "  ${DOCKER_REGISTRY}/microservices-tasks:${TAG}"
echo "  ${DOCKER_REGISTRY}/microservices-client:${TAG}"

# Function to print section headers
print_header() {
    echo -e "\n${BLUE}======================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}======================================${NC}"
}

# Function to print success message
print_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

# Function to print error message
print_error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

# Function to print info message
print_info() {
    echo -e "${YELLOW}[INFO] $1${NC}"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    print_success "Docker is running"
}

# Function to check if logged into Docker Hub
check_docker_login() {
    print_info "Checking Docker login status..."
    if ! docker login --username="${DOCKER_REGISTRY}" --password-stdin < /dev/null 2>&1 | grep -q "Login Succeeded"; then
        print_info "Please login to Docker Hub..."
        if ! docker login; then
            print_error "Docker login failed"
            exit 1
        fi
    fi
    print_success "Authenticated with Docker registry"
}

# Function to build and push a service
build_and_push() {
    local service_name=$1
    local dockerfile_path=$2
    local context_path=$3
    local build_args=$4
    
    print_header "Building and Pushing: ${service_name}"
    
    local image_name="${DOCKER_REGISTRY}/${service_name}:${TAG}"
    local image_latest="${DOCKER_REGISTRY}/${service_name}:latest"
    
    print_info "Image: ${image_name}"
    print_info "Context: ${context_path}"
    print_info "Dockerfile: ${dockerfile_path}"
    
    # Check if Dockerfile exists
    if [ ! -f "${dockerfile_path}" ]; then
        print_error "Dockerfile not found: ${dockerfile_path}"
        return 1
    fi
    
    # Check if context exists
    if [ ! -d "${context_path}" ]; then
        print_error "Context directory not found: ${context_path}"
        return 1
    fi
    
    # Build the image
    print_info "Building image..."
    local build_cmd="docker build"
    
    if [ -n "$build_args" ]; then
        print_info "Build arguments: ${build_args}"
        build_cmd="${build_cmd} ${build_args}"
    fi
    
    build_cmd="${build_cmd} -t ${image_name} -t ${image_latest} -f ${dockerfile_path} ${context_path}"
    
    if eval $build_cmd; then
        print_success "Build completed"
    else
        print_error "Build failed"
        return 1
    fi
    
    # Show image details
    local image_size=$(docker images "${image_name}" --format "{{.Size}}")
    local image_id=$(docker images "${image_name}" --format "{{.ID}}")
    print_info "Image ID: ${image_id}"
    print_info "Image Size: ${image_size}"
    
    # Push tagged version
    print_info "Pushing ${image_name}..."
    if docker push "${image_name}"; then
        print_success "Pushed ${image_name}"
    else
        print_error "Push failed for ${image_name}"
        return 1
    fi
    
    # Push latest tag if not already latest
    if [ "${TAG}" != "latest" ]; then
        print_info "Pushing ${image_latest}..."
        if docker push "${image_latest}"; then
            print_success "Pushed ${image_latest}"
        else
            print_error "Push failed for ${image_latest}"
            return 1
        fi
    fi
    
    print_success "Successfully built and pushed ${service_name}"
    return 0
}

# Function to build Auth Service
build_auth() {
    build_and_push "microservices-auth" "./auth/Dockerfile" "./auth" ""
}

# Function to build Tasks Service
build_tasks() {
    build_and_push "microservices-tasks" "./tasks/Dockerfile" "./tasks" ""
}

# Function to build Client (with environment selection)
build_client() {
    local env_type=${1:-k8s}
    local build_args=""
    
    case $env_type in
        dev|local)
            build_args="--build-arg VITE_AUTH_SERVICE_URL=http://localhost:3001/api/auth --build-arg VITE_TASKS_SERVICE_URL=http://localhost:3002/api/tasks"
            ;;
        k8s|kubernetes)
            build_args="--build-arg VITE_AUTH_SERVICE_URL=http://auth-service/api/auth --build-arg VITE_TASKS_SERVICE_URL=http://tasks-service/api/tasks"
            ;;
        k8s-fqdn)
            build_args="--build-arg VITE_AUTH_SERVICE_URL=http://auth-service.default.svc.cluster.local/api/auth --build-arg VITE_TASKS_SERVICE_URL=http://tasks-service.default.svc.cluster.local/api/tasks"
            ;;
        *)
            print_error "Unknown environment type: ${env_type}"
            print_info "Available types: dev, k8s, k8s-fqdn"
            return 1
            ;;
    esac
    
    build_and_push "microservices-client" "./client/Dockerfile" "./client" "${build_args}"
}

# Main script starts here
print_header "Microservices Docker Build and Push"

print_info "Docker Registry: ${DOCKER_REGISTRY}"
print_info "Service: ${SERVICE}"
print_info "Tag: ${TAG}"

# Check prerequisites
check_docker
check_docker_login

# Build counter
TOTAL=0
SUCCESS=0
FAILED=0

# Build services based on selection
case $SERVICE in
    all)
        print_info "Building all services..."
        TOTAL=3
        
        if build_auth; then ((SUCCESS++)); else ((FAILED++)); fi
        if build_tasks; then ((SUCCESS++)); else ((FAILED++)); fi
        
        # Ask for client environment type
        echo -e "\n${YELLOW}Select client build environment:${NC}"
        echo "  1) dev/local (localhost URLs)"
        echo "  2) k8s (Kubernetes short names)"
        echo "  3) k8s-fqdn (Kubernetes FQDN)"
        read -p "Enter choice [1-3] (default: 2): " client_env_choice
        
        case $client_env_choice in
            1) CLIENT_ENV="dev" ;;
            3) CLIENT_ENV="k8s-fqdn" ;;
            *) CLIENT_ENV="k8s" ;;
        esac
        
        if build_client "$CLIENT_ENV"; then ((SUCCESS++)); else ((FAILED++)); fi
        ;;
        
    auth)
        print_info "Building Auth Service only..."
        TOTAL=1
        if build_auth; then ((SUCCESS++)); else ((FAILED++)); fi
        ;;
        
    tasks)
        print_info "Building Tasks Service only..."
        TOTAL=1
        if build_tasks; then ((SUCCESS++)); else ((FAILED++)); fi
        ;;
        
    client)
        print_info "Building Client only..."
        TOTAL=1
        
        # Ask for environment type
        echo -e "\n${YELLOW}Select client build environment:${NC}"
        echo "  1) dev/local (localhost URLs)"
        echo "  2) k8s (Kubernetes short names)"
        echo "  3) k8s-fqdn (Kubernetes FQDN)"
        read -p "Enter choice [1-3] (default: 2): " client_env_choice
        
        case $client_env_choice in
            1) CLIENT_ENV="dev" ;;
            3) CLIENT_ENV="k8s-fqdn" ;;
            *) CLIENT_ENV="k8s" ;;
        esac
        
        if build_client "$CLIENT_ENV"; then ((SUCCESS++)); else ((FAILED++)); fi
        ;;
        
    *)
        print_error "Unknown service: ${SERVICE}"
        echo -e "\n${YELLOW}Usage: $0 [service] [tag]${NC}"
        echo -e "  service: all, auth, tasks, client (default: all)"
        echo -e "  tag: version tag (default: latest)"
        echo -e "\n${YELLOW}Examples:${NC}"
        echo -e "  $0                    # Build all services with latest tag"
        echo -e "  $0 all v1.0.0        # Build all services with v1.0.0 tag"
        echo -e "  $0 auth              # Build only auth service"
        echo -e "  $0 client            # Build only client"
        exit 1
        ;;
esac

# Print summary
print_header "Build Summary"
echo -e "${BLUE}Total services: ${TOTAL}${NC}"
echo -e "${GREEN}Successful: ${SUCCESS}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"

if [ $FAILED -eq 0 ]; then
    print_success "All builds completed successfully!"
    echo -e "\n${YELLOW}Images pushed to registry:${NC}"
    if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "auth" ]; then
        echo -e "  ${DOCKER_REGISTRY}/microservices-auth:${TAG}"
        [ "${TAG}" != "latest" ] && echo -e "  ${DOCKER_REGISTRY}/microservices-auth:latest"
    fi
    if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "tasks" ]; then
        echo -e "  ${DOCKER_REGISTRY}/microservices-tasks:${TAG}"
        [ "${TAG}" != "latest" ] && echo -e "  ${DOCKER_REGISTRY}/microservices-tasks:latest"
    fi
    if [ "$SERVICE" = "all" ] || [ "$SERVICE" = "client" ]; then
        echo -e "  ${DOCKER_REGISTRY}/microservices-client:${TAG}"
        [ "${TAG}" != "latest" ] && echo -e "  ${DOCKER_REGISTRY}/microservices-client:latest"
    fi
    echo -e "\n${YELLOW}Next steps:${NC}"
    echo -e "  1. Update image references in k8s deployment files"
    echo -e "  2. Deploy: kubectl apply -f k8s/"
    echo -e "  3. Verify: kubectl get pods"
    exit 0
else
    print_error "Some builds failed. Please check the errors above."
    exit 1
fi