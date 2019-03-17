pipeline {
  agent none
  environment {
    PROJECT_NAME = 'me-transactions'    
    APP_NAME = 'me_transactions'
    INSTANCE_TYPE = 'n1-highmem-8'    
  }
  stages {
    stage('Prepare instance') {
      agent {
        kubernetes {
          label 'create-instance'
          defaultContainer 'jnlp'
          instanceCap '4'
        }
      }
      steps {
        container(name: 'gcloud', shell: '/bin/sh') {
          sh 'apk update && apk add curl bash'
          sh 'env'
          withCredentials([file(credentialsId: 'e7e3e6df-8ef5-4738-a4d5-f56bb02a8bb2', variable: 'KEYFILE')]) {
            sh 'gcloud auth activate-service-account jenkins-pool@ehealth-162117.iam.gserviceaccount.com --key-file=${KEYFILE} --project=ehealth-162117'
            sh 'curl -s https://raw.githubusercontent.com/edenlabllc/ci-utils/umbrella_jenkins_new/create_instance.sh -o create_instance.sh; bash ./create_instance.sh'
          }
          slackSend (color: '#8E24AA', message: "Instance for ${GIT_URL[19..-5]}@$GIT_BRANCH created")
        }
      }
      post { 
        success {
          slackSend (color: 'good', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> (<${GIT_URL[0..-5]}/commit/$GIT_COMMIT|${GIT_COMMIT.take(7)}>) of ${GIT_URL[19..-5]}@$GIT_BRANCH by $GIT_COMMITTER_NAME STARTED")
        }
        failure {
          slackSend (color: 'danger', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> (<${GIT_URL[0..-5]}/commit/$GIT_COMMIT|${GIT_COMMIT.take(7)}>) of ${GIT_URL[19..-5]}@$GIT_BRANCH by $GIT_COMMITTER_NAME FAILED to start")
        }
        aborted {
          slackSend (color: 'warning', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> (<${GIT_URL[0..-5]}/commit/$GIT_COMMIT|${GIT_COMMIT.take(7)}>) of ${GIT_URL[19..-5]}@$GIT_BRANCH by $GIT_COMMITTER_NAME ABORTED before start")
        }
      }
    }
    stage('Build') {
      agent {
        kubernetes {
          label 'me-trans-build'
          defaultContainer 'jnlp'
          yaml '''
apiVersion: v1
kind: Pod
metadata:
  labels:
    stage: build
spec:
  tolerations:
  - key: "node"
    operator: "Equal"
    value: "ci"
    effect: "NoSchedule"
  containers:
  - name: docker
    image: lakone/docker:18.09-alpine3.9
    resources:
      limits:
        cpu: 1
        memory: 2048Mi
      requests:
        cpu: 200m
        memory: 256Mi     
    volumeMounts:
    - mountPath: /var/run/docker.sock
      name: volume
    command:
    - cat
    tty: true
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "384Mi"
        cpu: "500m"
  nodeSelector:
    cloud.google.com/gke-nodepool: $PROJECT_NAME
  volumes:
  - name: volume
    hostPath:
      path: /var/run/docker.sock
'''
        }
      }
      steps {
        container(name: 'docker', shell: '/bin/sh') {
          sh 'docker build --tag "edenlabllc/me_transactions:$GIT_COMMIT" --build-arg APP_NAME=${APP_NAME} .'
        }
      }
      post {
        always {
          container(name: 'docker', shell: '/bin/sh') {
            sh 'echo " ---- step: Remove docker image from host ---- ";'
            sh 'docker rmi edenlabllc/me_transactions:$GIT_COMMIT'
          }
        }
      }
    }
    stage('Deploy') {
      when {
        branch 'develop'
      }
      environment {
        APP_NAME = 'me_transactions'
      }
      agent {
        kubernetes {
          label 'me-trans-build'
          defaultContainer 'jnlp'
          yaml '''
apiVersion: v1
kind: Pod
metadata:
  labels:
    stage: build
spec:
  containers:
  - name: docker
    image: lakone/docker:18.09-alpine3.9    
    volumeMounts:
    - mountPath: /var/run/docker.sock
      name: volume
    command:
    - cat
    tty: true
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "384Mi"
        cpu: "500m"
  - name: kubectl
    image: lachlanevenson/k8s-kubectl:v1.13.2
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
      requests:
        cpu: 200m
        memory: 256Mi     
    command:
    - cat
    tty: true
  nodeSelector:
    node: ci
  volumes:
  - name: volume
    hostPath:
      path: /var/run/docker.sock
  nodeSelector:
    cloud.google.com/gke-nodepool: $PROJECT_NAME       
'''
        }
      }
      steps {
        container(name: 'docker', shell: '/bin/sh') {
          sh 'docker build --tag "edenlabllc/me_transactions:develop" --build-arg APP_NAME=${APP_NAME} .'
          withCredentials(bindings: [usernamePassword(credentialsId: '8232c368-d5f5-4062-b1e0-20ec13b0d47b', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_PASSWORD')]) {
            sh 'echo " ---- step: Push docker image ---- ";'
            sh 'echo "Logging in into Docker Hub";'
            sh 'echo ${DOCKER_PASSWORD} | docker login -u ${DOCKER_USERNAME} --password-stdin'
            sh 'docker push edenlabllc/me_transactions:develop'
          }
        }
        container(name: 'kubectl', shell: '/bin/sh') {
          sh 'kubectl delete pod -n me -l app=me-transactions'
        }
      }
      post {
        always {
          container(name: 'docker', shell: '/bin/sh') {
            sh 'echo " ---- step: Remove docker image from host ---- ";'
            sh 'docker rmi edenlabllc/me_transactions:develop'
          }
        }
      }
    }
  }
  post {
    success {
      slackSend (color: 'good', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> of ${JOB_NAME} passed in ${currentBuild.durationString}")
    }
    failure {
      slackSend (color: 'danger', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> of ${JOB_NAME} failed in ${currentBuild.durationString}")
    }
    aborted {
      slackSend (color: 'warning', message: "Build <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> of ${JOB_NAME} canceled in ${currentBuild.durationString}")
    }
    always {
      node('delete-instance') {
        container(name: 'gcloud', shell: '/bin/sh') {
          withCredentials([file(credentialsId: 'e7e3e6df-8ef5-4738-a4d5-f56bb02a8bb2', variable: 'KEYFILE')]) {
            checkout scm
            sh 'apk update && apk add curl bash'
            sh 'gcloud auth activate-service-account jenkins-pool@ehealth-162117.iam.gserviceaccount.com --key-file=${KEYFILE} --project=ehealth-162117'
            sh 'curl -s https://raw.githubusercontent.com/edenlabllc/ci-utils/umbrella_jenkins_new/delete_instance.sh -o delete_instance.sh; bash ./delete_instance.sh'
          }
          slackSend (color: '#4286F5', message: "Stage for deleting instance for job <${RUN_CHANGES_DISPLAY_URL[0..-8]}status|#$BUILD_NUMBER> passed")
        }
      }
    }
  }
}
