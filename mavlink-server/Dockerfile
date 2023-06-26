FROM python:3.10-alpine


COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

# Install dependencies

COPY main.py /app/script.py
WORKDIR /app
CMD ["python3", "main.py"]