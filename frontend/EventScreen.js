import React, { useState, useEffect } from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ActivityIndicator } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import FormModule from './components/modules/FormModule';

const API_URL = 'http://localhost:8080';

export default function EventScreen({ event, onNavigate, userId }) {
    const [currentEvent, setCurrentEvent] = useState(event);
    const [activeModule, setActiveModule] = useState(null);

    useEffect(() => {
        if (!event?.id) return;

        // Poll for event updates every 3 seconds
        const interval = setInterval(async () => {
            try {
                const response = await fetch(`${API_URL}/events/${event.id}`);
                if (response.ok) {
                    const data = await response.json();
                    setCurrentEvent(data);

                    // Check for active module
                    if (data.modules) {
                        const active = data.modules.find(m => m.status === 'active');
                        // Only update if changed to avoid re-renders
                        setActiveModule(prev => {
                            if (prev?.id === active?.id && prev?.status === active?.status) return prev;
                            return active || null;
                        });
                    }
                }
            } catch (error) {
                console.error('Polling error:', error);
            }
        }, 3000);

        return () => clearInterval(interval);
    }, [event?.id]);

    if (!currentEvent) {
        return (
            <View style={styles.container}>
                <Text style={styles.errorText}>No event data found.</Text>
                <TouchableOpacity onPress={() => onNavigate('main')} style={styles.backButton}>
                    <Text style={styles.backButtonText}>Go Back</Text>
                </TouchableOpacity>
            </View>
        );
    }

    return (
        <View style={styles.container}>
            <StatusBar style="light" />
            <View style={styles.content}>
                <View style={styles.header}>
                    <Text style={styles.statusText}>Connected to</Text>
                    <Text style={styles.eventName}>{currentEvent.name}</Text>
                </View>

                {activeModule ? (
                    <View style={styles.moduleContainer}>
                        {activeModule.type === 'form' && (
                            <FormModule
                                module={activeModule}
                                eventId={currentEvent.id}
                                userId={userId}
                            />
                        )}
                        {/* Add other module types here */}
                    </View>
                ) : (
                    <View style={styles.card}>
                        <ActivityIndicator size="large" color="#007AFF" style={styles.loader} />
                        <Text style={styles.message}>Waiting for host to start the next activity...</Text>
                        <Text style={styles.subMessage}>Sit tight and get ready!</Text>
                    </View>
                )}

                <TouchableOpacity
                    style={styles.leaveButton}
                    onPress={() => onNavigate('main')}
                >
                    <Text style={styles.leaveButtonText}>Leave Event</Text>
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#121212', // Dark theme matching admin
        alignItems: 'center',
        justifyContent: 'center',
    },
    content: {
        width: '90%',
        maxWidth: 400,
        alignItems: 'center',
    },
    header: {
        marginBottom: 40,
        alignItems: 'center',
    },
    statusText: {
        color: '#888',
        fontSize: 14,
        textTransform: 'uppercase',
        letterSpacing: 1,
        marginBottom: 8,
    },
    eventName: {
        color: '#FFF',
        fontSize: 32,
        fontWeight: '800',
        textAlign: 'center',
    },
    card: {
        backgroundColor: '#1E1E1E',
        borderRadius: 20,
        padding: 40,
        width: '100%',
        alignItems: 'center',
        marginBottom: 40,
        borderWidth: 1,
        borderColor: '#333',
    },
    moduleContainer: {
        width: '100%',
        marginBottom: 40,
    },
    loader: {
        marginBottom: 20,
    },
    message: {
        color: '#FFF',
        fontSize: 18,
        fontWeight: '600',
        textAlign: 'center',
        marginBottom: 8,
    },
    subMessage: {
        color: '#888',
        fontSize: 14,
        textAlign: 'center',
    },
    leaveButton: {
        paddingVertical: 16,
        paddingHorizontal: 32,
        borderRadius: 12,
        borderWidth: 1,
        borderColor: '#333',
        width: '100%',
        alignItems: 'center',
    },
    leaveButtonText: {
        color: '#FF453A',
        fontSize: 16,
        fontWeight: '600',
    },
    errorText: {
        color: '#FF453A',
        fontSize: 18,
        marginBottom: 20,
    },
    backButton: {
        padding: 10,
    },
    backButtonText: {
        color: '#FFF',
        fontSize: 16,
    },
});
