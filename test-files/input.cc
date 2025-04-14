// #include <tops.h>
// #include <tcle.h>
// #include <tops_runtime.h>
// #include <krt/mmu.h>

// #ifdef __CLION_IDE__
// #define __vector
// #define __vector2
// #define __vector4
// #define __vector8
// #endif
// using namespace tops;
const char * addr = "hello world";
  int index[sizeof(__vector int)/4];
// template<int N>
// __global__ void test_gather(char* from) {
//   int index[sizeof(__vector int)/4];
//   for (int i = 0; i < sizeof(__vector int)/4; i++) {
//     index[i] = i * 4;
//   }
//   auto addr = tops::map_mem((generic_ptr)from, N);
//   __vector int off = *reinterpret_cast<__vector int*>(index);
//   auto v = tcle::gather<__vector int>((__TCLE_AS__ void*)addr, off);
//   tcle::scatter(v, (__TCLE_AS__ void*)addr, off);
//   tops::unmap_mem(addr);
// }