-- 原子双删正向+反向索引
-- KEYS[1] = client_affinity:{groupID}:{clientID}       (正向)
-- KEYS[2] = client_affinity_rev:{groupID}:{accountID}  (反向)
-- ARGV[1] = accountID (正向索引中要删除的成员)
-- ARGV[2] = clientID  (反向索引中要删除的成员)
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[2])
return 1
