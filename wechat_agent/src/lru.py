from collections import OrderedDict
from typing import TypeVar, Generic, Optional

K = TypeVar('K')
V = TypeVar('V')

class LRUCache(Generic[K, V]):
    def __init__(self, capacity: int):
        self.capacity = capacity
        self.cache: OrderedDict[K, V] = OrderedDict()

    def get(self, key: K) -> Optional[V]:
        if key in self.cache:
            self.cache.move_to_end(key)
            return self.cache[key]
        return None

    def put(self, key: K, value: V) -> None:
        self.cache[key] = value
        self.cache.move_to_end(key)
        if len(self.cache) > self.capacity:
            self.cache.popitem(last=False)

    def __contains__(self, key: K) -> bool:
        return key in self.cache
