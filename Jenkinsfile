pipeline {
  agent none
  stages {
    stage('Build') {
      when {
        not {
          branch 'develop'
        }
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
  tolerations:
  - key: "node"
    operator: "Equal"
    value: "ci"
    effect: "NoSchedule"
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
  nodeSelector:
    node: ci
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
    stage('Build and deploy') {
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
  tolerations:
  - key: "node"
    operator: "Equal"
    value: "ci"
    effect: "NoSchedule"
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
    command:
    - cat
    tty: true
  nodeSelector:
    node: ci
  volumes:
  - name: volume
    hostPath:
      path: /var/run/docker.sock
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
      slackSend (color: 'good', message: "SUCCESSFUL: Job - ${env.JOB_NAME} ${env.BUILD_NUMBER} (<${env.BUILD_URL}|Open>) success in ${currentBuild.durationString}")
    }
    failure {
      slackSend (color: 'danger', message: "FAILED: Job - ${env.JOB_NAME} ${env.BUILD_NUMBER} (<${env.BUILD_URL}|Open>) failed in ${currentBuild.durationString}")
    }
    aborted {
      slackSend (color: 'warning', message: "ABORTED: Job - ${env.JOB_NAME} ${env.BUILD_NUMBER} (<${env.BUILD_URL}|Open>) canceled in ${currentBuild.durationString}")
    }
  }
}
