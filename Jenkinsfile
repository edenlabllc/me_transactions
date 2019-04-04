def author() {
  return sh(returnStdout: true, script: 'git log -n 1 --format="%an"').trim()
}
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
      script {
        if (env.CHANGE_ID == null) {
          slackSend (color: 'good', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/commit/${env.GIT_COMMIT}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} by ${author()} *success* in ${currentBuild.durationString.replace(' and counting', '')}")
        } else if (env.BRANCH_NAME.startsWith('PR')) {
          slackSend (color: 'good', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/pull/${env.CHANGE_ID}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} in PR #${env.CHANGE_ID} by ${author()} *success* in ${currentBuild.durationString.replace(' and counting', '')}")
        }
      }
    }
    failure {
      script {
        if (env.CHANGE_ID == null) {
          slackSend (color: 'danger', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/commit/${env.GIT_COMMIT}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} by ${author()} *failed* in ${currentBuild.durationString.replace(' and counting', '')}")
        } else if (env.BRANCH_NAME.startsWith('PR')) {
          slackSend (color: 'danger', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/pull/${env.CHANGE_ID}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} in PR #${env.CHANGE_ID} by ${author()} *failed* in ${currentBuild.durationString.replace(' and counting', '')}")
        }
      }
    }
    aborted {
      script {
        if (env.CHANGE_ID == null) {
          slackSend (color: 'warning', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/commit/${env.GIT_COMMIT}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} by ${author()} *canceled* in ${currentBuild.durationString.replace(' and counting', '')}")
        } else if (env.BRANCH_NAME.startsWith('PR')) {
          slackSend (color: 'warning', message: "Build <${env.RUN_DISPLAY_URL}|#${env.BUILD_NUMBER}> (<https://github.com/edenlabllc/ehealth.api/pull/${env.CHANGE_ID}|${env.GIT_COMMIT.take(7)}>) of ${env.JOB_NAME} in PR #${env.CHANGE_ID} by ${author()} *canceled* in ${currentBuild.durationString.replace(' and counting', '')}")
        }
      }
    }
  }
}

