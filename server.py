from flask import Flask, jsonify
import hashlib
import base64

app = Flask(__name__)

# Sample data
temp_storage = {}

# GET /urls - Get all urls
@app.route('/urls', methods=['GET'])
def get_urls():
    return jsonify({'urls': urls})

# GET /shorten/<curr_url> - Get a single url by ID
@app.route('/shorten/<curr_url>', methods=['GET'])
def get_url(curr_url):
    sha256_hash = hashlib.sha256(curr_url.encode()).digest()
    base64_encoded = base64.urlsafe_b64encode(sha256_hash).decode()[:6]  # Taking only the first 6 characters
    if base64_encoded not in temp_storage:
        temp_storage[curr_url] = base64_encoded
    return jsonify({'shortened': base64_encoded})

# POST
#@app.route('/post', methods=['POST'])


if __name__ == '__main__':
    app.run(debug=True)
    print("aaa")