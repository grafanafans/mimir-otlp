# Mimir otlp 

A example about mimir with OTLP.

## How to run

```
git clone git@github.com:grafanafans/mimir-otlp.git
cd mimir-otlp && docker-compose up -d
```

batch send requests to generate metrics

```
wrk -d 1m http://localhost:8080/users/1 
```

At last jump to Grafana with link `http://localhost:3000` and visit the default dashboard, you will see:

![otlp-mimir](https://user-images.githubusercontent.com/1459834/190450498-73916699-a8a4-4d86-bb23-a733e3028a45.png)
