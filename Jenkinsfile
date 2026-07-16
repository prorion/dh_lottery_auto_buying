pipeline {
  agent any

  environment {
    APP_DIR = '/opt/dhlottery'
  }

  options {
    timestamps()
    disableConcurrentBuilds()
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Check config') {
      steps {
        sh '''
          set -e
          if [ ! -f "${APP_DIR}/config.json" ]; then
            echo "ERROR: ${APP_DIR}/config.json 이 없습니다."
            echo "서버에 config.json을 먼저 생성하세요."
            exit 1
          fi
        '''
      }
    }

    stage('Link config & logs') {
      steps {
        sh '''
          set -e
          mkdir -p "${APP_DIR}/logs"
          ln -sfn "${APP_DIR}/config.json" "${WORKSPACE}/config.json"
          ln -sfn "${APP_DIR}/logs" "${WORKSPACE}/logs"
        '''
      }
    }

    stage('Deploy') {
      steps {
        sh '''
          set -e
          docker compose build
          docker compose up -d --remove-orphans
          docker compose ps
        '''
      }
    }

    stage('Smoke check') {
      steps {
        sh '''
          set -e
          sleep 2
          docker ps --filter "name=dhlottery" --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
          docker logs --tail 50 dhlottery || true
        '''
      }
    }
  }

  post {
    success {
      echo '배포 완료: dhlottery 컨테이너 기동됨'
    }
    failure {
      echo '배포 실패 — Console Output / docker logs dhlottery 확인'
    }
  }
}
