#if __GCU_ARCH__ == 400
int gcu400_value;
#elif defined(__EFGCU_ARCH__)
int efgcu_value;
#else
int unknown_value;
#endif
