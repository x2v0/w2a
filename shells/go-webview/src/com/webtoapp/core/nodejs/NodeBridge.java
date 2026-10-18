package com.webtoapp.core.nodejs;

import android.util.Log;

public final class NodeBridge {
    private static final String TAG = "NodeBridge";

    static {
        try {
            System.loadLibrary("node_bridge");
            Log.i(TAG, "libnode_bridge.so loaded");
        } catch (UnsatisfiedLinkError e) {
            Log.e(TAG, "libnode_bridge.so load failed: " + e.getMessage());
        }
    }

    public interface OutputCallback {
        void onOutput(String line, boolean isError);
    }

    public static native boolean nativeLoadNode(String nodePath);
    public static native int nativeStartNode(String[] arguments, Object callback);
    public static native boolean nativeIsStarted();
    public static native boolean nativeIsLoaded();

    private NodeBridge() {}
}
