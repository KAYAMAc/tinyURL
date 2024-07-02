FROM tiangolo/uwsgi-nginx-flask:python3.11

WORKDIR /app

#RUN pip install --no-cache-dir -r requirements.txt

ENV FLASK_APP=server.py

CMD ["flask", "run", "--host=0.0.0.0"]