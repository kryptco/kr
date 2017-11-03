#ifndef KRBTLE_H_
#define KRBTLE_H_

#include <stdlib.h>
int krbtle_add_service(char* service_uuid, unsigned long long len);
int krbtle_remove_service(char* service_uuid, unsigned long long len);
int krbtle_stop();
int krbtle_write_data(char* service_uuid, unsigned long long len,
                      uint8_t* data, unsigned long long data_len);

typedef void(*KRBTLE_ON_BLUETOOTH_DATA_T)(const void *, unsigned long long);
int krbtle_set_on_bluetooth_data(KRBTLE_ON_BLUETOOTH_DATA_T*);

#endif
