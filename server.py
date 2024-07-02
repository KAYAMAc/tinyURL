from flask import Flask, jsonify
import hashlib
import base64

# Sample data

app = Flask(__name__)

temp_storage = {}

# GET /urls - Get all urls
@app.route('/urls', methods=['GET'])
def get_urls():
    sql = "SELECT * FROM url"
    data = db.select_db(sql)
    return jsonify({"code": 0, "data": data, "msg": "查询成功"})

# GET /shorten/<curr_url> - Get a single url by ID
@app.route('/shorten/<curr_url>', methods=['GET'])
def get_url(curr_url):
    sha256_hash = hashlib.sha256(curr_url.encode()).digest()
    base64_encoded = base64.urlsafe_b64encode(sha256_hash).decode()[:6]  # Taking only the first 6 characters
    if base64_encoded not in temp_storage:
        temp_storage[curr_url] = base64_encoded
    return jsonify({'shortened': base64_encoded})

@app.route('/access/<short_url>', methods=['GET'])
def redirect(short_url):
    sha256_hash = hashlib.sha256(curr_url.encode()).digest()
    base64_encoded = base64.urlsafe_b64encode(sha256_hash).decode()[:6]  # Taking only the first 6 characters
    if base64_encoded not in temp_storage:
        temp_storage[curr_url] = base64_encoded
    return jsonify({'shortened': base64_encoded})

if __name__ == '__main__':
    app.run(host='0.0.0.0')

# POST
#@app.route('/post', methods=['POST'])


#app.run(debug=True)
#print("aaa")


