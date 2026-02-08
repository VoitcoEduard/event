import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, TextInput, Alert, ActivityIndicator, Platform } from 'react-native';
import { StatusBar } from 'expo-status-bar';

const API_URL = 'http://localhost:8080';

export default function MainScreen({ onNavigate, onJoinEvent }) {
    const [pin, setPin] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    useEffect(() => {
        if (Platform.OS === 'web' && typeof window !== 'undefined') {
            const params = new URLSearchParams(window.location.search);
            const pinParam = params.get('pin');
            if (pinParam) {
                setPin(pinParam);
            }
        }
    }, []);

    const handleConnect = async () => {
        setError('');
        if (!pin || pin.length !== 6) {
            setError('Please enter a valid 6-digit PIN');
            return;
        }

        setLoading(true);
        try {
            const response = await fetch(`${API_URL}/events/connect`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ pin }),
            });

            if (response.ok) {
                const event = await response.json();
                onJoinEvent(event);
            } else {
                const errorData = await response.text();
                // Check for specific error message or default
                if (response.status === 404) {
                    setError('Event not found. Check your PIN.');
                } else if (response.status === 403) {
                    setError('This event is not active currently.');
                } else {
                    setError('Failed to connect to event.');
                }
            }
        } catch (error) {
            console.error('Connection error:', error);
            setError('Network error. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    return (
        <View style={styles.container}>
            <StatusBar style="dark" />
            <View style={styles.content}>
                <Text style={styles.title}>Hello from Expo</Text>
                <Text style={styles.subtitle}>Minimalist Design</Text>

                <View style={styles.card}>
                    <Text style={styles.cardTitle}>Connect to Event</Text>
                    <Text style={styles.cardText}>
                        Enter the 6-digit PIN code to join the live event.
                    </Text>
                    <TextInput
                        style={styles.input}
                        placeholder="000000"
                        placeholderTextColor="#CCC"
                        keyboardType="number-pad"
                        maxLength={6}
                        value={pin}
                        onChangeText={(text) => {
                            setPin(text);
                            setError('');
                        }}
                    />
                    {error ? <Text style={styles.errorText}>{error}</Text> : null}
                    <TouchableOpacity
                        style={styles.connectButton}
                        onPress={handleConnect}
                        disabled={loading}
                    >
                        {loading ? (
                            <ActivityIndicator color="#FFF" />
                        ) : (
                            <Text style={styles.connectButtonText}>Connect</Text>
                        )}
                    </TouchableOpacity>
                </View>

                <TouchableOpacity
                    style={styles.adminLink}
                    onPress={() => onNavigate('admin')}
                >
                    <Text style={styles.adminLinkText}>Go to Admin Dashboard</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#FAFAFA',
        alignItems: 'center',
        justifyContent: 'center',
    },
    content: {
        width: '90%',
        maxWidth: 400,
        alignItems: 'center',
    },
    title: {
        fontSize: 32,
        fontWeight: '800',
        color: '#1A1A1A',
        marginBottom: 8,
        letterSpacing: -0.5,
    },
    subtitle: {
        fontSize: 16,
        color: '#666666',
        marginBottom: 40,
        fontWeight: '500',
        letterSpacing: 0.5,
        textTransform: 'uppercase',
    },
    card: {
        backgroundColor: '#FFFFFF',
        padding: 32,
        borderRadius: 24,
        width: '100%',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 8 },
        shadowOpacity: 0.08,
        shadowRadius: 24,
        elevation: 8,
        marginBottom: 32,
        borderWidth: 1,
        borderColor: '#F0F0F0',
        alignItems: 'center',
    },
    cardTitle: {
        fontSize: 20,
        fontWeight: '700',
        color: '#1A1A1A',
        marginBottom: 8,
    },
    cardText: {
        fontSize: 15,
        color: '#666',
        textAlign: 'center',
        marginBottom: 24,
        lineHeight: 22,
    },
    input: {
        backgroundColor: '#F5F5F7',
        width: '100%',
        borderRadius: 12,
        padding: 16,
        fontSize: 24,
        fontWeight: '700',
        textAlign: 'center',
        letterSpacing: 8,
        marginBottom: 24,
        color: '#1A1A1A',
    },
    connectButton: {
        backgroundColor: '#007AFF',
        paddingVertical: 16,
        borderRadius: 12,
        width: '100%',
        alignItems: 'center',
        shadowColor: '#007AFF',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.2,
        shadowRadius: 8,
    },
    connectButtonText: {
        color: '#FFFFFF',
        fontSize: 18,
        fontWeight: '600',
    },
    adminLink: {
        padding: 16,
    },
    adminLinkText: {
        color: '#666',
        fontSize: 14,
        fontWeight: '500',
    },
    errorText: {
        color: '#FF3B30',
        fontSize: 14,
        marginBottom: 16,
        textAlign: 'center',
        fontWeight: '500',
    },
});
