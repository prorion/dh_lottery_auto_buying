pipeline {
  agent any

  environment {
    // Docker 호스트 경로 (Jenkins 컨테이너 내부 경로가 아님)
    APP_DIR = '/opt/dhlottery'
  }

  options {
    timestamps()
    disableConcurrentBuilds()
    // SCM 은 Job 설정(Pipeline from SCM)에서 이미 checkout 함
    skipDefaultCheckout(false)
  }

  stages {
    stage('Check config on host') {
      steps {
        sh '''
          set -e
          # Jenkins 가 컨테이너여도, docker.sock 으로 호스트 파일을 확인
          if ! docker run --rm -v "${APP_DIR}:${APP_DIR}:ro" alpine:3.21 test -f "${APP_DIR}/config.json"; then
            echo "ERROR: 호스트에 ${APP_DIR}/config.json 이 없습니다."
            echo "Docker 호스트에서 아래를 실행하세요:"
            echo "  sudo mkdir -p ${APP_DIR}/logs"
            echo "  sudo nano ${APP_DIR}/config.json"
            echo "  sudo chmod 600 ${APP_DIR}/config.json"
            exit 1
          fi
          docker run --rm -v "${APP_DIR}:${APP_DIR}" alpine:3.21 mkdir -p "${APP_DIR}/logs"
          echo "config.json OK"
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
