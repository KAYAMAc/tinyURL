import redis
from config.setting import REDIS_HOST, REDIS_PORT, REDIS_PASSWD, EXPIRE_TIME


class RedisDb():

    def __init__(self, host, port, passwd):
        self.r = redis.Redis(
            host=host,
            port=port,
            password=passwd,
            decode_responses=True 
        )

    def handle_redis_token(self, key, value=None):
        if value: 
            self.r.set(key, value, ex=EXPIRE_TIME)
        else: 
            redis_token = self.r.get(key)
            return redis_token


redis_db = RedisDb(REDIS_HOST, REDIS_PORT, REDIS_PASSWD)