pipeline {
  agent any

  environment {
    // Docker 호스트 경로 (Jenkins 컨테이너 내부 경로가 아님)
    APP_DIR = '/opt/dhlottery'
  }

  options {
    timestamps()
    disableConcurrentBuilds()
    skipDefaultCheckout(false)
  }

  stages {
    stage('Check config on host') {
      steps {
        sh '''
          set -e
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

          # docker compose(v2 플러그인) 또는 docker-compose(v1) 중 사용 가능한 것 선택
          if docker compose version >/dev/null 2>&1; then
            COMPOSE="docker compose"
          elif command -v docker-compose >/dev/null 2>&1; then
            COMPOSE="docker-compose"
          else
            echo "ERROR: docker compose / docker-compose 를 찾을 수 없습니다."
            echo "Jenkins 컨테이너에 둘 중 하나를 설치하세요."
            echo "  예) apk add docker-cli-compose   # alpine"
            echo "  예) apt-get install -y docker-compose-plugin"
            docker version || true
            exit 1
          fi

          echo "Using: ${COMPOSE}"
          ${COMPOSE} build
          ${COMPOSE} up -d --remove-orphans
          ${COMPOSE} ps
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
