FROM openjdk:8-jre-alpine

ARG GIT_REPOSITORY=
ARG GIT_BRANCH_NAME=
ARG GIT_COMMIT_ID=
ARG ARTIFACT_VERSION=1.0-SNAPSHOT

ADD dashboards-controller/target/dashboards-controller-$ARTIFACT_VERSION.jar ./app.jar

ENTRYPOINT [ "java", "-jar", "app.jar" ]

LABEL GIT_REPOSITORY=$GIT_REPOSITORY \
      GIT_BRANCH_NAME=$GIT_BRANCH_NAME \
      GIT_COMMIT_ID=$GIT_COMMIT_ID \
      ARTIFACT_VERSION=$ARTIFACT_VERSION
      