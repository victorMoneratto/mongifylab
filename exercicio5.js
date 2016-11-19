// Without index
// 100 inserts: 0.091s
// 100 finds: 0.001s
// 1000 inserts: 1.03s
// 1000 finds: 0.011s
// 10000 inserts: 8.855s
// 10000 finds: 0.106s
// 100000 inserts: 90.894s
// 100000 finds: 1.201s

// With index
// 100 inserts: 0.876s
// 100 finds: 0.001s
// 1000 inserts: 8.654s
// 1000 finds: 0.01s
// 10000 inserts: 87.35s
// 10000 finds: 0.12s
// 100000 inserts: 883.417s
// 100000 finds: 1.163s

function randString() {
    var str = "";
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

    for(var i=0; i < 2000; i++) {
        str += chars.charAt(Math.floor(Math.random() * chars.length));
    }

    return str;
}

var cache = [];
const CACHE_MAX = 100;

function insertOne() {
    var obj = {};
    for(var i = 1; i <= 10; i++) {
        obj["param"+i] = randString();
    }

    if (cache.length < CACHE_MAX) {
        cache.push(obj.param1);
    }

    db.benchmark.insert(obj);
}

function findOne() {
    var query = {
        param1: cache[Math.floor(Math.random() * cache.length)] 
    } 

    db.benchmark.find(query);
}



if(typeof db !== 'undefined') {
    
    for (var times = 100; times <= 100000; times *= 10) {
        cache = []; 
        db.createCollection('benchmark');
        // db.benchmark.createIndex({param1: 1});

        var times = 100;
        var insertStart = new Date();
        for (var i = 0; i < times; i++) {
            insertOne();
        }
        var insertEnd = new Date();
        print(times + " inserts: " + (insertEnd - insertStart) / 1000 + "s");

        times = 100000;
        var findStart = new Date();
        for (var i = 0; i < times; i++) {
            findOne();
        }
        var findEnd = new Date();
        print(times + " finds: " + (findEnd - findStart) / 1000 + "s");

        db.benchmark.drop();
    }
}