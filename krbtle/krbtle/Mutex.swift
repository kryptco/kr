/*
 * Sync (sync.swift) - Please be Safe
 *
 * Copyright (C) 2015 ONcast, LLC. All Rights Reserved.
 * Created by Josh Baker (joshbaker77@gmail.com)
 *
 * This software may be modified and distributed under the terms
 * of the MIT license.  See the LICENSE file for details.
 *
 * Portions of the documentation of this code are reproduced from
 * work created and shared by Google and used according to terms
 * described in the Creative Commons 3.0 Attribution License.
 *
 * http://golang.org/pkg/sync/
 */

import Foundation

/// A Mutex is a mutual exclusion lock.
public class Mutex {
    private var mutex = pthread_mutex_t()
    /// Returns a new Mutex.
    public init(){
        pthread_mutex_init(&mutex, nil)
    }
    deinit{
        pthread_mutex_destroy(&mutex)
    }
    /// Locks the mutex. If the lock is already in use, the calling operation blocks until the mutex is available.
    public func lock(){
        pthread_mutex_lock(&mutex)
    }
    /**
     Unlocks the mutex. It's an undefined error if mutex is not locked on entry to unlock.
     
     A locked Mutex is not associated with a particular operation.
     It is allowed for one operation to lock a Mutex and then arrange for another operation to unlock it.
     */
    public func unlock(){
        pthread_mutex_unlock(&mutex)
    }
    
    /// Locks the mutex before calling the function. Unlocks after closure is completed
    /// - Parameter: closure Closure function
    public func lock(closure : ()->()) {
        lock()
        closure()
        unlock()
    }
}
