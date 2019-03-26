pipeline {
  agent {
    node { 
      label 'ehealth-build' 
    }
  }
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
      steps {
          sh 'sudo docker build --tag "edenlabllc/me_transactions:$GIT_COMMIT" --build-arg APP_NAME=${APP_NAME} .'
      }
      post {
        always {
            sh 'echo " ---- step: Remove docker image from host ---- ";'
            sh 'sudo docker rmi edenlabllc/me_transactions:$GIT_COMMIT'
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
      steps {
          sh 'sudo docker build --tag "edenlabllc/me_transactions:develop" --build-arg APP_NAME=${APP_NAME} .'
          withCredentials(bindings: [usernamePassword(credentialsId: '8232c368-d5f5-4062-b1e0-20ec13b0d47b', usernameVariable: 'DOCKER_USERNAME', passwordVariable: 'DOCKER_PASSWORD')]) {
            sh 'echo " ---- step: Push docker image ---- ";'
            sh 'echo "Logging in into Docker Hub";'
            sh 'echo ${DOCKER_PASSWORD} | sudo docker login -u ${DOCKER_USERNAME} --password-stdin'
            sh 'sudo docker push edenlabllc/me_transactions:develop'
          }
          withCredentials([file(credentialsId: '091bd05c-0219-4164-8a17-777f4caf7481', variable: 'GCLOUD_KEY')]) {
            sh '''
              gcloud auth activate-service-account --key-file=$GCLOUD_KEY
              gcloud container clusters get-credentials dev --zone europe-west1-d --project ehealth-162117
              kubectl delete pod -n me -l app=me-transactions
            '''
          }
        }
      post {
        always {
          sh 'echo " ---- step: Remove docker image from host ---- ";'
          sh 'sudo docker rmi edenlabllc/me_transactions:develop'
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

