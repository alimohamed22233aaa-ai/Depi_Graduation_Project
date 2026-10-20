FROM eclipse-temurin:21-jdk-jammy AS build

WORKDIR /src
COPY gradlew build.gradle settings.gradle ./
COPY gradle/wrapper/gradle-wrapper.jar gradle/wrapper/gradle-wrapper.properties ./gradle/wrapper/
RUN chmod +x gradlew
RUN ./gradlew --no-daemon dependencies
COPY src ./src
RUN ./gradlew --no-daemon installDist

FROM gcr.io/distroless/java21-debian12:nonroot
WORKDIR /app
COPY --from=build /src/build/install/hipstershop/lib ./lib
EXPOSE 9555
ENTRYPOINT ["/usr/bin/java", "-cp", "/app/lib/*", "hipstershop.AdService"]
