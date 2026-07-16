pipeline {
  agent any

  environment {
    // Docker 호스트 경로 (Jenkins 컨테이너 내부 경로가 아님)
    APP_DIR = '/opt/dhlottery'
    IMAGE = 'dhlottery:latest'
    CONTAINER = 'dhlottery'
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
        // Jenkins 컨테이너에 compose 플러그인이 없어도 동작 (docker CLI만 사용)
        sh '''
          set -e

          echo "Building ${IMAGE} ..."
          docker build -t "${IMAGE}" .

          echo "Recreating ${CONTAINER} ..."
          docker stop "${CONTAINER}" 2>/dev/null || true
          docker rm "${CONTAINER}" 2>/dev/null || true

          docker run -d \
            --name "${CONTAINER}" \
            --restart unless-stopped \
            -e TZ=Asia/Seoul \
            -v "${APP_DIR}/config.json:/app/config.json:ro" \
            -v "${APP_DIR}/logs:/app/logs" \
            "${IMAGE}" \
            -service

          docker ps --filter "name=${CONTAINER}"
        '''
      }
    }

    stage('Smoke check') {
      steps {
        sh '''
          set -e
          sleep 2
          docker ps --filter "name=${CONTAINER}" --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
          docker logs --tail 50 "${CONTAINER}" || true
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
