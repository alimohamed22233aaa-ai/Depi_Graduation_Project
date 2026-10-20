FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build

WORKDIR /src
COPY src/cartservice.csproj ./src/
RUN dotnet restore ./src/cartservice.csproj
COPY src ./src
RUN dotnet publish ./src/cartservice.csproj -c Release --no-restore -o /out --no-self-contained

FROM mcr.microsoft.com/dotnet/aspnet:10.0-alpine
WORKDIR /app
ENV ASPNETCORE_URLS=http://+:7070
COPY --from=build /out ./
EXPOSE 7070
USER app
ENTRYPOINT ["dotnet", "cartservice.dll"]
