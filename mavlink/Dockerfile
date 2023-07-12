FROM python:3.9

RUN mkdir /app
# Set the working directory
WORKDIR /app

# Copy the script and requirements.txt
COPY main.py .
COPY requirements.txt .
# Install dependencies
RUN pip install -r requirements.txt
# Define the command to run the script
CMD ["python3", "main.py"]
