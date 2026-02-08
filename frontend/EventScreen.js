import React from 'react';
import { StyleSheet, Text, View, TouchableOpacity, ActivityIndicator } from 'react-native';
import { StatusBar } from 'expo-status-bar';

export default function EventScreen({ event, onNavigate }) {
    if (!event) {
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
                    <Text style={styles.eventName}>{event.name}</Text>
                </View>

                <View style={styles.card}>
                    <ActivityIndicator size="large" color="#007AFF" style={styles.loader} />
                    <Text style={styles.message}>Waiting for host to start the game...</Text>
                    <Text style={styles.subMessage}>Sit tight and get ready!</Text>
                </View>

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
        backgroundColor: '#121212', // Dark theme matching admin but for user
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
