FROM python:3.14.2-slim

WORKDIR /usr/src/app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY docker-rate-limit.py ./

CMD [ "python3", "./docker-rate-limit.py" ]

